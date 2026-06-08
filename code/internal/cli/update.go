package cli

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/spf13/cobra"
)

const (
	githubRepo    = "tickplatform/tick"
	markerFile    = ".tick-source"
	updateTimeout = 60 * time.Second
)

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [version]",
		Short: "升级 tick 到新版本（支持 release / from-git / from-go 三种模式）",
		Long: `升级当前 tick 二进制。自动探测安装模式：
  - 二进制同目录存在 .tick-source  → from-git 模式（dev build）
  - 不存在 .tick-source             → release 模式（标准 install）

可使用 --from-release / --from-git / --from-go 强制指定模式。

示例:
  tick update --check               # 仅检查
  tick update                       # 自动模式升级
  tick update v0.3.0                # 升到指定版本
  tick update --from-git            # 强制从本地 git 仓库升级
  tick update --from-go             # 强制通过 go install 升级
  tick update --from-release --force  # 跳过版本比较强制升级`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			version := ""
			if len(args) > 0 {
				version = args[0]
			}
			check, _ := cmd.Flags().GetBool("check")
			fromRelease, _ := cmd.Flags().GetBool("from-release")
			fromGit, _ := cmd.Flags().GetBool("from-git")
			fromGo, _ := cmd.Flags().GetBool("from-go")
			force, _ := cmd.Flags().GetBool("force")

			runUpdate(version, check, fromRelease, fromGit, fromGo, force)
		},
	}
	cmd.Flags().Bool("check", false, "only check for updates; do not install")
	cmd.Flags().Bool("from-release", false, "force release mode (download from GitHub)")
	cmd.Flags().Bool("from-git", false, "force from-git mode (uses local repo + .tick-source)")
	cmd.Flags().Bool("from-go", false, "force from-go mode (go install)")
	cmd.Flags().Bool("force", false, "skip version comparison")
	return cmd
}

func runUpdate(targetVersion string, check, fromRel, fromGit, fromGo, force bool) {
	exe, err := os.Executable()
	if err != nil {
		exitErr("locate executable", err)
	}
	realExe, err := filepath.EvalSymlinks(exe)
	if err != nil {
		realExe = exe
	}
	exeDir := filepath.Dir(realExe)
	marker := filepath.Join(exeDir, markerFile)
	mode, sourcePath, err := detectUpdateMode(marker)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: 探测 .tick-source 失败: %v (fallback release)\n", err)
		mode = "release"
	}

	// flag overrides
	switch {
	case fromRel:
		mode = "release"
	case fromGit:
		mode = "from-git"
		if sourcePath == "" {
			exitErr("--from-git 需要 .tick-source marker", nil)
		}
	case fromGo:
		mode = "from-go"
	}

	fmt.Printf("当前版本: %s\n", buildVersion)
	fmt.Printf("模式:      %s\n", mode)
	if check {
		runCheck(mode, targetVersion)
		return
	}

	if !isWritable(exe) {
		fmt.Fprintf(os.Stderr, "需要 sudo 权限，请用 sudo env HOME=$HOME tick update\n")
		os.Exit(3)
	}

	switch mode {
	case "from-git":
		doUpdateFromGit(sourcePath, realExe)
	case "from-go":
		doUpdateFromGo(targetVersion, realExe)
	default:
		doUpdateRelease(targetVersion, force, realExe)
	}
}

// detectUpdateMode inspects the .tick-source marker.
// Returns ("from-git", sourcePath, nil) if marker is present and readable,
// ("release", "", nil) if not present.
func detectUpdateMode(markerPath string) (string, string, error) {
	data, err := os.ReadFile(markerPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "release", "", nil
		}
		return "release", "", err
	}
	p := strings.TrimSpace(string(data))
	if p == "" || !filepath.IsAbs(p) {
		return "release", p, fmt.Errorf(".tick-source 内容不是绝对路径: %q", p)
	}
	return "from-git", p, nil
}

func runCheck(mode, targetVersion string) {
	switch mode {
	case "from-git":
		if targetVersion == "" {
			fmt.Println("模式: from-git (本地仓库); --check 仅对 release 模式有意义")
			return
		}
		fmt.Printf("目标版本: %s (将使用 make build-dev 重新构建)\n", targetVersion)
	case "from-go":
		if targetVersion == "" {
			fmt.Println("模式: from-go; --check 仅对 release 模式有意义")
			return
		}
		fmt.Printf("目标版本: %s (将执行 go install @%s)\n", targetVersion, targetVersion)
	default:
		rel, ok, err := selfupdate.DetectVersion(githubRepo, targetVersion)
		if err != nil {
			exitErr("check update", err)
		}
		if !ok {
			fmt.Println("最新版本:", rel.Version.String())
			fmt.Println("需要更新: 否")
			return
		}
		fmt.Println("当前版本:", buildVersion)
		fmt.Println("最新版本:", rel.Version.String())
		fmt.Println("需要更新: 是")
	}
}

func doUpdateRelease(targetVersion string, force bool, exePath string) {
	fmt.Println("模式: release (从 GitHub Releases 下载)")
	rel, err := selfupdate.UpdateSelf(parseCurrentVersion(), githubRepo)
	if err != nil {
		// fallback: manual download + SHA256 verify
		if v, ok := parseVersion(targetVersion); ok {
			if err2 := manualReleaseUpdate(v, exePath); err2 != nil {
				exitErr("release update", fmt.Errorf("library err: %v; manual err: %w", err, err2))
			}
			return
		}
		exitErr("release update", err)
	}
	fmt.Printf("已升级到 %s（原二进制保留在 .old）\n", rel.Version.String())
}

func parseCurrentVersion() semver.Version {
	v, err := semver.Parse(strings.TrimPrefix(buildVersion, "v"))
	if err != nil {
		return semver.Version{Major: 0, Minor: 0, Patch: 0}
	}
	return v
}

func manualReleaseUpdate(version semver.Version, exePath string) error {
	assetName := fmt.Sprintf("tick-%s-%s", runtime.GOOS, runtime.GOARCH)
	url := fmt.Sprintf("https://github.com/%s/releases/download/v%s/%s", githubRepo, version.String(), assetName)
	shaURL := url + ".sha256"

	client := &http.Client{Timeout: updateTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download HTTP %d", resp.StatusCode)
	}
	tmp, err := os.CreateTemp("", "tick-update-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, h), resp.Body); err != nil {
		return err
	}
	_ = tmp.Close()

	// fetch sha256
	sum := hex.EncodeToString(h.Sum(nil))
	shaResp, err := client.Get(shaURL)
	if err == nil {
		defer shaResp.Body.Close()
		body, _ := io.ReadAll(shaResp.Body)
		expected := strings.TrimSpace(strings.Split(string(body), " ")[0])
		if expected != "" && !strings.EqualFold(expected, sum) {
			return fmt.Errorf("SHA256 校验失败: expected=%s got=%s (原二进制保留)", expected, sum)
		}
	} else {
		fmt.Fprintf(os.Stderr, "warning: 无法下载 .sha256 (%v)，跳过校验\n", err)
	}

	return atomicReplace(tmp.Name(), exePath)
}

func doUpdateFromGit(sourcePath, exePath string) {
	fmt.Printf("模式: from-git (源码: %s)\n", sourcePath)
	if _, err := os.Stat(filepath.Join(sourcePath, "go.mod")); err != nil {
		exitErr("源码目录无效", err)
	}

	// 1. git pull
	fmt.Println("→ git pull...")
	pull := exec.Command("git", "pull")
	pull.Dir = sourcePath
	pull.Stdout = os.Stdout
	pull.Stderr = os.Stderr
	if err := pull.Run(); err != nil {
		exitErr("git pull 失败（原二进制保留）", err)
	}

	// 2. resolve target version (HEAD commit short SHA)
	rev := exec.Command("git", "rev-parse", "--short", "HEAD")
	rev.Dir = sourcePath
	out, err := rev.Output()
	if err != nil {
		exitErr("git rev-parse 失败", err)
	}
	commit := strings.TrimSpace(string(out))
	fmt.Printf("→ make build-dev (commit %s)...\n", commit)

	// 3. make build-dev
	mk := exec.Command("make", "build-dev")
	mk.Dir = sourcePath
	mk.Stdout = os.Stdout
	mk.Stderr = os.Stderr
	if err := mk.Run(); err != nil {
		exitErr("make build-dev 失败（原二进制保留）", err)
	}

	// 4. atomic replace
	built := filepath.Join(sourcePath, "bin", "tick")
	if _, err := os.Stat(built); err != nil {
		exitErr("构建产物不存在", err)
	}
	if err := atomicReplace(built, exePath); err != nil {
		exitErr("替换二进制失败", err)
	}
	fmt.Println("已升级（原二进制保留在 .old）")
}

func doUpdateFromGo(targetVersion, exePath string) {
	if targetVersion == "" {
		targetVersion = "latest"
	}
	fmt.Printf("模式: from-go (go install @%s)\n", targetVersion)
	pkg := fmt.Sprintf("github.com/%s/cmd/tick@%s", githubRepo, targetVersion)
	cmd := exec.Command("go", "install", pkg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		exitErr("go install 失败（原二进制保留）", err)
	}
	// go install 默认输出到 $GOBIN/$GOPATH/bin/tick
	gobin := os.Getenv("GOBIN")
	if gobin == "" {
		home, _ := os.UserHomeDir()
		gobin = filepath.Join(home, "go", "bin")
	}
	built := filepath.Join(gobin, "tick")
	if _, err := os.Stat(built); err != nil {
		exitErr("go install 产物未找到", err)
	}
	if err := atomicReplace(built, exePath); err != nil {
		exitErr("替换二进制失败", err)
	}
	fmt.Println("已升级（原二进制保留在 .old）")
}

// atomicReplace replaces finalPath with newPath via 4-step mv sequence.
// Falls back to cp + rm on EXDEV (cross-volume).
func atomicReplace(newPath, finalPath string) error {
	oldBackup := finalPath + ".old"
	_ = os.Remove(oldBackup) // clean any leftover

	if err := os.Rename(finalPath, oldBackup); err != nil {
		if isCrossDeviceErr(err) {
			if err := copyFile(finalPath, oldBackup); err != nil {
				return fmt.Errorf("backup old (cp fallback): %w", err)
			}
			// finalPath is still there; we'll overwrite via cp below
		} else {
			return fmt.Errorf("backup old: %w", err)
		}
	}
	if err := os.Rename(newPath, finalPath); err != nil {
		if isCrossDeviceErr(err) {
			if err := copyFile(newPath, finalPath); err != nil {
				_ = os.Rename(oldBackup, finalPath)
				return fmt.Errorf("install new (cp fallback): %w", err)
			}
			_ = os.Remove(newPath)
		} else {
			_ = os.Rename(oldBackup, finalPath)
			return fmt.Errorf("install new: %w", err)
		}
	}
	if err := os.Chmod(finalPath, 0o755); err != nil {
		return err
	}
	_ = os.Remove(oldBackup)
	return nil
}

func isCrossDeviceErr(err error) bool {
	if err == nil {
		return false
	}
	// Go does not export a typed error; we use string match as a pragmatic signal.
	msg := err.Error()
	return strings.Contains(msg, "cross-device link") || strings.Contains(msg, "EXDEV")
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func isWritable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if info.IsDir() {
		return false
	}
	dir := filepath.Dir(path)
	test := filepath.Join(dir, ".tick-write-test")
	f, err := os.Create(test)
	if err != nil {
		return false
	}
	_ = f.Close()
	_ = os.Remove(test)
	return true
}

func parseVersion(s string) (semver.Version, bool) {
	if s == "" {
		return semver.Version{}, false
	}
	s = strings.TrimPrefix(s, "v")
	v, err := semver.Parse(s)
	if err != nil {
		return semver.Version{}, false
	}
	return v, true
}

// confirm prompts the user; auto-returns true when --yes is set.
func confirm(yes bool, prompt string) bool {
	if yes {
		return true
	}
	fmt.Fprintf(os.Stderr, "%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	ans := strings.ToLower(strings.TrimSpace(line))
	return ans == "y" || ans == "yes"
}
