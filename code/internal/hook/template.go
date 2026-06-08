package hook

import (
	"fmt"
	"regexp"
)

var tmplRegex = regexp.MustCompile(`\{\{([\w-]+)\}\}`)

func ExtractPlaceholders(s string) []string {
	matches := tmplRegex.FindAllStringSubmatch(s, -1)
	seen := make(map[string]bool)
	var result []string
	for _, m := range matches {
		if !seen[m[1]] {
			seen[m[1]] = true
			result = append(result, m[1])
		}
	}
	return result
}

func Render(template string, vc *VariableContext) (string, error) {
	var renderErr error
	result := tmplRegex.ReplaceAllStringFunc(template, func(match string) string {
		key := tmplRegex.FindStringSubmatch(match)[1]
		val, ok := vc.Get(key)
		if !ok {
			renderErr = fmt.Errorf("undefined variable: %s", key)
			return match
		}
		return val
	})
	if renderErr != nil {
		return "", renderErr
	}
	return result, nil
}

func RenderJSON(data []byte, vc *VariableContext) ([]byte, error) {
	rendered, err := Render(string(data), vc)
	if err != nil {
		return nil, err
	}
	return []byte(rendered), nil
}
