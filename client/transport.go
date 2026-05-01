package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

func newHTTPClient(config Config) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	if config.CACertFile != "" || config.InsecureSkipVerify {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: config.InsecureSkipVerify, //nolint:gosec // Explicit opt-in for private deployments.
		}

		if config.CACertFile != "" {
			rootCAs, err := x509.SystemCertPool()
			if err != nil {
				rootCAs = x509.NewCertPool()
			}
			cert, err := os.ReadFile(config.CACertFile)
			if err != nil {
				panic(fmt.Errorf("read LOGCHEF_CA_CERT_FILE: %w", err))
			}
			if ok := rootCAs.AppendCertsFromPEM(cert); !ok {
				panic(fmt.Errorf("LOGCHEF_CA_CERT_FILE does not contain any PEM certificates"))
			}
			tlsConfig.RootCAs = rootCAs
		}

		transport.TLSClientConfig = tlsConfig
	}

	return &http.Client{
		Timeout: config.Timeout,
		Transport: cloudflareAccessTransport{
			base:          transport,
			config:        config,
			tokenProvider: newCloudflareTokenProvider(config),
		},
	}
}

type cloudflareAccessTransport struct {
	base          http.RoundTripper
	config        Config
	tokenProvider *cloudflareTokenProvider
}

func (t cloudflareAccessTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.config.CloudflareAccessClientID != "" {
		req.Header.Set("CF-Access-Client-ID", t.config.CloudflareAccessClientID)
	}
	if t.config.CloudflareAccessSecret != "" {
		req.Header.Set("CF-Access-Client-Secret", t.config.CloudflareAccessSecret)
	}
	if t.config.CloudflareAccessToken != "" {
		req.Header.Set("cf-access-token", t.config.CloudflareAccessToken)
	} else if t.tokenProvider != nil {
		token, err := t.tokenProvider.Token(req.Context())
		if err != nil {
			return nil, err
		}
		if token != "" {
			req.Header.Set("cf-access-token", token)
		}
	}

	for _, cookie := range t.cloudflareCookies() {
		req.AddCookie(cookie)
	}

	return t.base.RoundTrip(req)
}

type cloudflareTokenProvider struct {
	appURL          string
	cloudflaredPath string
	autoLogin       bool
	mu              sync.Mutex
	token           string
	expiresAt       time.Time
}

func newCloudflareTokenProvider(config Config) *cloudflareTokenProvider {
	if config.CloudflareAccessAppURL == "" {
		return nil
	}
	cloudflaredPath := config.CloudflaredPath
	if cloudflaredPath == "" {
		cloudflaredPath = "cloudflared"
	}
	return &cloudflareTokenProvider{
		appURL:          config.CloudflareAccessAppURL,
		cloudflaredPath: cloudflaredPath,
		autoLogin:       config.CloudflareAccessAutoLogin,
	}
}

func (p *cloudflareTokenProvider) Token(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.token != "" && time.Until(p.expiresAt) > time.Minute {
		return p.token, nil
	}

	token, err := p.run(ctx, "token", "-app="+p.appURL)
	if err != nil && p.autoLogin {
		if _, loginErr := p.run(ctx, "login", p.appURL); loginErr != nil {
			return "", fmt.Errorf("cloudflared access login failed: %w", loginErr)
		}
		token, err = p.run(ctx, "token", "-app="+p.appURL)
	}
	if err != nil {
		return "", fmt.Errorf("cloudflared access token failed; run `cloudflared access login %s`: %w", p.appURL, err)
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return "", fmt.Errorf("cloudflared access token returned an empty token for %s", p.appURL)
	}

	p.token = token
	p.expiresAt = jwtExpiry(token)
	if p.expiresAt.IsZero() {
		p.expiresAt = time.Now().Add(5 * time.Minute)
	}
	return p.token, nil
}

func (p *cloudflareTokenProvider) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, p.cloudflaredPath, append([]string{"access"}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func jwtExpiry(token string) time.Time {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return time.Time{}
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp == 0 {
		return time.Time{}
	}
	return time.Unix(claims.Exp, 0)
}

func (t cloudflareAccessTransport) cloudflareCookies() []*http.Cookie {
	cookies := make([]*http.Cookie, 0, 2)
	if t.config.CloudflareAuthorizationJWT != "" {
		cookies = append(cookies, &http.Cookie{
			Name:  "CF_Authorization",
			Value: t.config.CloudflareAuthorizationJWT,
		})
	}
	if t.config.CloudflareAppSession != "" {
		cookies = append(cookies, &http.Cookie{
			Name:  "CF_AppSession",
			Value: t.config.CloudflareAppSession,
		})
	}
	return cookies
}

func ParseBoolEnv(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
