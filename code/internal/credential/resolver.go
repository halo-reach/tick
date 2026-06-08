package credential

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tidwall/gjson"
	"golang.org/x/sync/singleflight"
	"github.com/tickplatform/tick/internal/domain"
)

type ResolvedCredential struct {
	Name           string
	Code           string
	Type           domain.CredentialType
	Token          string
	Headers        map[string]string
	Secret         string
	InjectLocation *string
	InjectKey      *string
	InjectPrefix   *string
}

type Resolver struct {
	store  *Store
	cache  *TokenCache
	sf     singleflight.Group
	client *http.Client
}

func NewResolver(store *Store, cache *TokenCache) *Resolver {
	return &Resolver{
		store:  store,
		cache:  cache,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (r *Resolver) ResolveByCode(ctx context.Context, code, tenantID string) (*ResolvedCredential, error) {
	cred, config, err := r.store.GetDecryptedByCode(ctx, code, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get credential by code %q: %w", code, err)
	}
	return r.resolveFromCred(cred, config)
}

func (r *Resolver) Resolve(ctx context.Context, credID, tenantID string) (*ResolvedCredential, error) {
	cred, config, err := r.store.GetDecrypted(ctx, credID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get credential: %w", err)
	}
	return r.resolveFromCred(cred, config)
}

func (r *Resolver) resolveFromCred(cred *domain.Credential, config json.RawMessage) (*ResolvedCredential, error) {

	var injectCfg struct {
		InjectLocation *string `json:"inject_location"`
		InjectKey      *string `json:"inject_key"`
		InjectPrefix   *string `json:"inject_prefix"`
	}
	_ = json.Unmarshal(config, &injectCfg)

	base := ResolvedCredential{
		Name:           cred.Name,
		Code:           cred.Code,
		Type:           cred.Type,
		InjectLocation: injectCfg.InjectLocation,
		InjectKey:      injectCfg.InjectKey,
		InjectPrefix:   injectCfg.InjectPrefix,
	}

	switch cred.Type {
	case domain.CredTypeBearer:
		var cfg struct {
			Token string `json:"token"`
		}
		if err := json.Unmarshal(config, &cfg); err != nil {
			return nil, err
		}
		base.Token = cfg.Token
		return &base, nil

	case domain.CredTypeBasic:
		var cfg struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.Unmarshal(config, &cfg); err != nil {
			return nil, err
		}
		base.Token = base64.StdEncoding.EncodeToString([]byte(cfg.Username + ":" + cfg.Password))
		return &base, nil

	case domain.CredTypeCustomHeader:
		var cfg struct {
			Headers map[string]string `json:"headers"`
		}
		if err := json.Unmarshal(config, &cfg); err != nil {
			return nil, err
		}
		base.Headers = cfg.Headers
		return &base, nil

	case domain.CredTypeHMAC:
		var cfg struct {
			Secret string `json:"secret"`
		}
		if err := json.Unmarshal(config, &cfg); err != nil {
			return nil, err
		}
		base.Secret = cfg.Secret
		return &base, nil

	case domain.CredTypeDynamic:
		token, err := r.resolveDynamic(context.Background(), cred.ID, config)
		if err != nil {
			return nil, err
		}
		base.Token = token
		return &base, nil

	case domain.CredTypeOAuth2CC:
		token, err := r.resolveOAuth2CC(context.Background(), cred.ID, config)
		if err != nil {
			return nil, err
		}
		base.Token = token
		return &base, nil

	default:
		return nil, fmt.Errorf("unsupported credential type: %s", cred.Type)
	}
}


func (r *Resolver) resolveDynamic(ctx context.Context, credID string, config json.RawMessage) (string, error) {
	if cached, err := r.cache.Get(ctx, credID); err == nil {
		return cached, nil
	} else if err != redis.Nil {
		// log but continue
	}

	val, err, _ := r.sf.Do(credID, func() (any, error) {
		return r.fetchDynamicToken(ctx, config)
	})
	if err != nil {
		// Retry once
		val, err, _ = r.sf.Do(credID+":retry", func() (any, error) {
			return r.fetchDynamicToken(ctx, config)
		})
		if err != nil {
			return "", fmt.Errorf("dynamic token fetch failed: %w", err)
		}
	}

	result := val.(*dynamicResult)
	if result.ttl > 0 {
		_ = r.cache.Set(ctx, credID, result.token, result.ttl)
	}
	return result.token, nil
}

type dynamicResult struct {
	token string
	ttl   time.Duration
}

func (r *Resolver) fetchDynamicToken(ctx context.Context, config json.RawMessage) (*dynamicResult, error) {
	var cfg struct {
		TokenRequest struct {
			URL     string            `json:"url"`
			Method  string            `json:"method"`
			Headers map[string]string `json:"headers"`
			Body    json.RawMessage   `json:"body"`
		} `json:"token_request"`
		TokenExtract struct {
			Path string `json:"path"`
			TTL  int    `json:"ttl_secs"`
		} `json:"token_extract"`
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, err
	}

	method := cfg.TokenRequest.Method
	if method == "" {
		method = "POST"
	}

	var bodyReader io.Reader
	if cfg.TokenRequest.Body != nil {
		body := string(cfg.TokenRequest.Body)
		// If body is a JSON string (quoted), unwrap it
		var str string
		if json.Unmarshal(cfg.TokenRequest.Body, &str) == nil {
			body = str
		}
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, cfg.TokenRequest.URL, bodyReader)
	if err != nil {
		return nil, err
	}
	for k, v := range cfg.TokenRequest.Headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("Content-Type") == "" && bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("token request returned HTTP %d", resp.StatusCode)
	}

	token := gjson.GetBytes(respBody, cfg.TokenExtract.Path).String()
	if token == "" {
		return nil, fmt.Errorf("token not found at path %q", cfg.TokenExtract.Path)
	}

	ttl := time.Duration(cfg.TokenExtract.TTL) * time.Second
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &dynamicResult{token: token, ttl: ttl}, nil
}

func (r *Resolver) resolveOAuth2CC(ctx context.Context, credID string, config json.RawMessage) (string, error) {
	if cached, err := r.cache.Get(ctx, credID); err == nil {
		return cached, nil
	} else if err != redis.Nil {
		// log but continue
	}

	val, err, _ := r.sf.Do(credID, func() (any, error) {
		return r.fetchOAuth2Token(ctx, config)
	})
	if err != nil {
		val, err, _ = r.sf.Do(credID+":retry", func() (any, error) {
			return r.fetchOAuth2Token(ctx, config)
		})
		if err != nil {
			return "", fmt.Errorf("oauth2 token fetch failed: %w", err)
		}
	}

	result := val.(*dynamicResult)
	if result.ttl > 0 {
		_ = r.cache.Set(ctx, credID, result.token, result.ttl)
	}
	return result.token, nil
}

func (r *Resolver) fetchOAuth2Token(ctx context.Context, config json.RawMessage) (*dynamicResult, error) {
	var cfg struct {
		TokenURL     string `json:"token_url"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		Scope        string `json:"scope"`
		TTL          int    `json:"ttl_secs"`
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, err
	}

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
	}
	if cfg.Scope != "" {
		form.Set("scope", cfg.Scope)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", cfg.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("oauth2 token request returned HTTP %d", resp.StatusCode)
	}

	token := gjson.GetBytes(respBody, "access_token").String()
	if token == "" {
		return nil, fmt.Errorf("access_token not found in response")
	}

	ttl := time.Duration(cfg.TTL) * time.Second
	if ttl <= 0 {
		if expiresIn := gjson.GetBytes(respBody, "expires_in").Int(); expiresIn > 0 {
			ttl = time.Duration(float64(expiresIn)*0.9) * time.Second
		} else {
			ttl = 5 * time.Minute
		}
	}
	return &dynamicResult{token: token, ttl: ttl}, nil
}
