package mcplogchef

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/server"

	"github.com/mr-karan/logchef-mcp/client"
)

const (
	defaultLogchefHost = "localhost:5173"
	defaultLogchefURL  = "http://" + defaultLogchefHost

	logchefURLEnvVar = "LOGCHEF_URL"
	logchefAPIEnvVar = "LOGCHEF_API_KEY"

	logchefCACertFileEnvVar           = "LOGCHEF_CA_CERT_FILE"
	logchefInsecureSkipVerifyEnvVar   = "LOGCHEF_INSECURE_SKIP_VERIFY"
	logchefCFAccessClientIDEnvVar     = "LOGCHEF_CF_ACCESS_CLIENT_ID"
	logchefCFAccessClientSecretEnvVar = "LOGCHEF_CF_ACCESS_CLIENT_SECRET"
	logchefCFAccessTokenEnvVar        = "LOGCHEF_CF_ACCESS_TOKEN"
	logchefCFAccessAppURLEnvVar       = "LOGCHEF_CF_ACCESS_APP_URL"
	logchefCloudflaredPathEnvVar      = "LOGCHEF_CLOUDFLARED_PATH"
	logchefCFAccessAutoLoginEnvVar    = "LOGCHEF_CF_ACCESS_AUTO_LOGIN"
	logchefCFAuthorizationEnvVar      = "LOGCHEF_CF_AUTHORIZATION"
	logchefCFAppSessionEnvVar         = "LOGCHEF_CF_APPSESSION"
	logchefURLHeader                  = "X-Logchef-URL"
	logchefAPIKeyHeader               = "X-Logchef-API-Key"
	logchefCACertFileHeader           = "X-Logchef-CA-Cert-File"
	logchefInsecureSkipVerifyHeader   = "X-Logchef-Insecure-Skip-Verify"
	logchefCFAccessClientIDHeader     = "X-Logchef-CF-Access-Client-ID"
	logchefCFAccessClientSecretHeader = "X-Logchef-CF-Access-Client-Secret"
	logchefCFAccessTokenHeader        = "X-Logchef-CF-Access-Token"
	logchefCFAccessAppURLHeader       = "X-Logchef-CF-Access-App-URL"
	logchefCloudflaredPathHeader      = "X-Logchef-Cloudflared-Path"
	logchefCFAccessAutoLoginHeader    = "X-Logchef-CF-Access-Auto-Login"
	logchefCFAuthorizationHeader      = "X-Logchef-CF-Authorization"
	logchefCFAppSessionHeader         = "X-Logchef-CF-AppSession"
)

type logchefClientConfig struct {
	URL                  string
	APIKey               string
	CACertFile           string
	InsecureSkipVerify   bool
	CFAccessClientID     string
	CFAccessClientSecret string
	CFAccessToken        string
	CFAccessAppURL       string
	CloudflaredPath      string
	CFAccessAutoLogin    bool
	CFAuthorizationJWT   string
	CFAppSession         string
}

func logchefConfigFromEnv() logchefClientConfig {
	return logchefClientConfig{
		URL:                  strings.TrimRight(os.Getenv(logchefURLEnvVar), "/"),
		APIKey:               os.Getenv(logchefAPIEnvVar),
		CACertFile:           os.Getenv(logchefCACertFileEnvVar),
		InsecureSkipVerify:   client.ParseBoolEnv(os.Getenv(logchefInsecureSkipVerifyEnvVar)),
		CFAccessClientID:     os.Getenv(logchefCFAccessClientIDEnvVar),
		CFAccessClientSecret: os.Getenv(logchefCFAccessClientSecretEnvVar),
		CFAccessToken:        os.Getenv(logchefCFAccessTokenEnvVar),
		CFAccessAppURL:       os.Getenv(logchefCFAccessAppURLEnvVar),
		CloudflaredPath:      os.Getenv(logchefCloudflaredPathEnvVar),
		CFAccessAutoLogin:    client.ParseBoolEnv(os.Getenv(logchefCFAccessAutoLoginEnvVar)),
		CFAuthorizationJWT:   os.Getenv(logchefCFAuthorizationEnvVar),
		CFAppSession:         os.Getenv(logchefCFAppSessionEnvVar),
	}
}

func logchefConfigFromHeaders(req *http.Request) logchefClientConfig {
	return logchefClientConfig{
		URL:                  strings.TrimRight(req.Header.Get(logchefURLHeader), "/"),
		APIKey:               req.Header.Get(logchefAPIKeyHeader),
		CACertFile:           req.Header.Get(logchefCACertFileHeader),
		InsecureSkipVerify:   client.ParseBoolEnv(req.Header.Get(logchefInsecureSkipVerifyHeader)),
		CFAccessClientID:     req.Header.Get(logchefCFAccessClientIDHeader),
		CFAccessClientSecret: req.Header.Get(logchefCFAccessClientSecretHeader),
		CFAccessToken:        req.Header.Get(logchefCFAccessTokenHeader),
		CFAccessAppURL:       req.Header.Get(logchefCFAccessAppURLHeader),
		CloudflaredPath:      req.Header.Get(logchefCloudflaredPathHeader),
		CFAccessAutoLogin:    client.ParseBoolEnv(req.Header.Get(logchefCFAccessAutoLoginHeader)),
		CFAuthorizationJWT:   req.Header.Get(logchefCFAuthorizationHeader),
		CFAppSession:         req.Header.Get(logchefCFAppSessionHeader),
	}
}

func (c logchefClientConfig) withFallback(fallback logchefClientConfig) logchefClientConfig {
	if c.URL == "" {
		c.URL = fallback.URL
	}
	if c.APIKey == "" {
		c.APIKey = fallback.APIKey
	}
	if c.CACertFile == "" {
		c.CACertFile = fallback.CACertFile
	}
	if !c.InsecureSkipVerify {
		c.InsecureSkipVerify = fallback.InsecureSkipVerify
	}
	if c.CFAccessClientID == "" {
		c.CFAccessClientID = fallback.CFAccessClientID
	}
	if c.CFAccessClientSecret == "" {
		c.CFAccessClientSecret = fallback.CFAccessClientSecret
	}
	if c.CFAccessToken == "" {
		c.CFAccessToken = fallback.CFAccessToken
	}
	if c.CFAccessAppURL == "" {
		c.CFAccessAppURL = fallback.CFAccessAppURL
	}
	if c.CloudflaredPath == "" {
		c.CloudflaredPath = fallback.CloudflaredPath
	}
	if !c.CFAccessAutoLogin {
		c.CFAccessAutoLogin = fallback.CFAccessAutoLogin
	}
	if c.CFAuthorizationJWT == "" {
		c.CFAuthorizationJWT = fallback.CFAuthorizationJWT
	}
	if c.CFAppSession == "" {
		c.CFAppSession = fallback.CFAppSession
	}
	return c
}

type logchefURLKey struct{}
type logchefAPIKeyKey struct{}

// logchefDebugKey is the context key for the Logchef transport's debug flag.
type logchefDebugKey struct{}

// WithLogchefDebug adds the Logchef debug flag to the context.
func WithLogchefDebug(ctx context.Context, debug bool) context.Context {
	if debug {
		slog.Info("Logchef transport debug mode enabled")
	}
	return context.WithValue(ctx, logchefDebugKey{}, debug)
}

// LogchefDebugFromContext extracts the Logchef debug flag from the context.
// If the flag is not set, it returns false.
func LogchefDebugFromContext(ctx context.Context) bool {
	if debug, ok := ctx.Value(logchefDebugKey{}).(bool); ok {
		return debug
	}
	return false
}

// ExtractLogchefInfoFromEnv is a StdioContextFunc that extracts Logchef configuration
// from environment variables and injects the configuration into the context.
var ExtractLogchefInfoFromEnv server.StdioContextFunc = func(ctx context.Context) context.Context {
	config := logchefConfigFromEnv()
	if config.URL == "" {
		config.URL = defaultLogchefURL
	}
	parsedURL, err := url.Parse(config.URL)
	if err != nil {
		panic(fmt.Errorf("invalid Logchef URL %s: %w", config.URL, err))
	}
	slog.Info("Using Logchef configuration", "url", parsedURL.Redacted(), "api_key_set", config.APIKey != "")
	return WithLogchefURL(WithLogchefAPIKey(ctx, config.APIKey), config.URL)
}

// httpContextFunc is a function that can be used as a `server.HTTPContextFunc` or a
// `server.SSEContextFunc`. It is necessary because, while the two types are functionally
// identical, they have distinct types and cannot be passed around interchangeably.
type httpContextFunc func(ctx context.Context, req *http.Request) context.Context

// ExtractLogchefInfoFromHeaders is a HTTPContextFunc that extracts Logchef configuration
// from request headers and injects the configuration into the context.
var ExtractLogchefInfoFromHeaders httpContextFunc = func(ctx context.Context, req *http.Request) context.Context {
	config := logchefConfigFromHeaders(req).withFallback(logchefConfigFromEnv())
	if config.URL == "" {
		config.URL = defaultLogchefURL
	}
	return WithLogchefURL(WithLogchefAPIKey(ctx, config.APIKey), config.URL)
}

// WithLogchefURL adds the Logchef URL to the context.
func WithLogchefURL(ctx context.Context, url string) context.Context {
	return context.WithValue(ctx, logchefURLKey{}, url)
}

// WithLogchefAPIKey adds the Logchef API key to the context.
func WithLogchefAPIKey(ctx context.Context, apiKey string) context.Context {
	return context.WithValue(ctx, logchefAPIKeyKey{}, apiKey)
}

// LogchefURLFromContext extracts the Logchef URL from the context.
func LogchefURLFromContext(ctx context.Context) string {
	if u, ok := ctx.Value(logchefURLKey{}).(string); ok {
		return u
	}
	return defaultLogchefURL
}

// LogchefAPIKeyFromContext extracts the Logchef API key from the context.
func LogchefAPIKeyFromContext(ctx context.Context) string {
	if k, ok := ctx.Value(logchefAPIKeyKey{}).(string); ok {
		return k
	}
	return ""
}

type logchefClientKey struct{}

// NewLogchefClient creates a Logchef client with the provided URL and API key.
func NewLogchefClient(ctx context.Context, logchefURL, apiKey string) *client.Client {
	config := logchefConfigFromEnv()
	config.URL = logchefURL
	config.APIKey = apiKey
	return NewLogchefClientFromConfig(ctx, config)
}

func NewLogchefClientFromConfig(ctx context.Context, config logchefClientConfig) *client.Client {
	logchefURL := config.URL
	apiKey := config.APIKey
	if logchefURL == "" {
		logchefURL = defaultLogchefURL
	}

	parsedURL, err := url.Parse(logchefURL)
	if err != nil {
		panic(fmt.Errorf("invalid Logchef URL: %w", err))
	}

	slog.Debug("Creating Logchef client", "url", parsedURL.Redacted(), "api_key_set", apiKey != "")
	return client.New(client.Config{
		BaseURL:                    logchefURL,
		APIKey:                     apiKey,
		CACertFile:                 config.CACertFile,
		InsecureSkipVerify:         config.InsecureSkipVerify,
		CloudflareAccessClientID:   config.CFAccessClientID,
		CloudflareAccessSecret:     config.CFAccessClientSecret,
		CloudflareAccessToken:      config.CFAccessToken,
		CloudflareAccessAppURL:     config.CFAccessAppURL,
		CloudflaredPath:            config.CloudflaredPath,
		CloudflareAccessAutoLogin:  config.CFAccessAutoLogin,
		CloudflareAuthorizationJWT: config.CFAuthorizationJWT,
		CloudflareAppSession:       config.CFAppSession,
	})
}

// ExtractLogchefClientFromEnv is a StdioContextFunc that extracts Logchef configuration
// from environment variables and injects a configured client into the context.
var ExtractLogchefClientFromEnv server.StdioContextFunc = func(ctx context.Context) context.Context {
	// Extract transport config from env vars
	config := logchefConfigFromEnv()
	if config.URL == "" {
		config.URL = defaultLogchefURL
	}

	logchefClient := NewLogchefClientFromConfig(ctx, config)
	return context.WithValue(ctx, logchefClientKey{}, logchefClient)
}

// ExtractLogchefClientFromHeaders is a HTTPContextFunc that extracts Logchef configuration
// from request headers and injects a configured client into the context.
var ExtractLogchefClientFromHeaders httpContextFunc = func(ctx context.Context, req *http.Request) context.Context {
	// Extract transport config from request headers, and set it on the context.
	config := logchefConfigFromHeaders(req).withFallback(logchefConfigFromEnv())
	if config.URL == "" {
		config.URL = defaultLogchefURL
	}

	logchefClient := NewLogchefClientFromConfig(ctx, config)
	return WithLogchefClient(ctx, logchefClient)
}

// WithLogchefClient sets the Logchef client in the context.
//
// It can be retrieved using LogchefClientFromContext.
func WithLogchefClient(ctx context.Context, client *client.Client) context.Context {
	return context.WithValue(ctx, logchefClientKey{}, client)
}

// LogchefClientFromContext retrieves the Logchef client from the context.
func LogchefClientFromContext(ctx context.Context) *client.Client {
	c, ok := ctx.Value(logchefClientKey{}).(*client.Client)
	if !ok {
		return nil
	}
	return c
}

// ComposeStdioContextFuncs composes multiple StdioContextFuncs into a single one.
func ComposeStdioContextFuncs(funcs ...server.StdioContextFunc) server.StdioContextFunc {
	return func(ctx context.Context) context.Context {
		for _, f := range funcs {
			ctx = f(ctx)
		}
		return ctx
	}
}

// ComposeSSEContextFuncs composes multiple SSEContextFuncs into a single one.
func ComposeSSEContextFuncs(funcs ...httpContextFunc) server.SSEContextFunc {
	return func(ctx context.Context, req *http.Request) context.Context {
		for _, f := range funcs {
			ctx = f(ctx, req)
		}
		return ctx
	}
}

// ComposeHTTPContextFuncs composes multiple HTTPContextFuncs into a single one.
func ComposeHTTPContextFuncs(funcs ...httpContextFunc) server.HTTPContextFunc {
	return func(ctx context.Context, req *http.Request) context.Context {
		for _, f := range funcs {
			ctx = f(ctx, req)
		}
		return ctx
	}
}

// ComposedStdioContextFunc returns a StdioContextFunc that comprises all predefined StdioContextFuncs,
// as well as the Logchef debug flag.
func ComposedStdioContextFunc(debug bool) server.StdioContextFunc {
	return ComposeStdioContextFuncs(
		func(ctx context.Context) context.Context {
			return WithLogchefDebug(ctx, debug)
		},
		ExtractLogchefInfoFromEnv,
		ExtractLogchefClientFromEnv,
	)
}

// ComposedSSEContextFunc is a SSEContextFunc that comprises all predefined SSEContextFuncs.
func ComposedSSEContextFunc(debug bool) server.SSEContextFunc {
	return ComposeSSEContextFuncs(
		func(ctx context.Context, req *http.Request) context.Context {
			return WithLogchefDebug(ctx, debug)
		},
		ExtractLogchefInfoFromHeaders,
		ExtractLogchefClientFromHeaders,
	)
}

// ComposedHTTPContextFunc is a HTTPContextFunc that comprises all predefined HTTPContextFuncs.
func ComposedHTTPContextFunc(debug bool) server.HTTPContextFunc {
	return ComposeHTTPContextFuncs(
		func(ctx context.Context, req *http.Request) context.Context {
			return WithLogchefDebug(ctx, debug)
		},
		ExtractLogchefInfoFromHeaders,
		ExtractLogchefClientFromHeaders,
	)
}
