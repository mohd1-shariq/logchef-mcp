package client

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCloudflareAccessTransportAddsHeadersAndCookies(t *testing.T) {
	var sawRequest bool
	transport := cloudflareAccessTransport{
		base: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			sawRequest = true
			if got := r.Header.Get("CF-Access-Client-ID"); got != "client-id" {
				t.Fatalf("CF-Access-Client-ID = %q", got)
			}
			if got := r.Header.Get("CF-Access-Client-Secret"); got != "client-secret" {
				t.Fatalf("CF-Access-Client-Secret = %q", got)
			}
			if got := r.Header.Get("cf-access-token"); got != "access-token" {
				t.Fatalf("cf-access-token = %q", got)
			}
			if cookie, err := r.Cookie("CF_Authorization"); err != nil || cookie.Value != "jwt" {
				t.Fatalf("CF_Authorization cookie = %v, %v", cookie, err)
			}
			if cookie, err := r.Cookie("CF_AppSession"); err != nil || cookie.Value != "session" {
				t.Fatalf("CF_AppSession cookie = %v, %v", cookie, err)
			}
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		}),
		config: Config{
			CloudflareAccessClientID:   "client-id",
			CloudflareAccessSecret:     "client-secret",
			CloudflareAccessToken:      "access-token",
			CloudflareAuthorizationJWT: "jwt",
			CloudflareAppSession:       "session",
		},
	}

	req, err := http.NewRequest(http.MethodGet, "https://logchef.example.test/api/v1/me", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if _, err := transport.RoundTrip(req); err != nil {
		t.Fatalf("round trip failed: %v", err)
	}

	if !sawRequest {
		t.Fatal("base transport did not receive request")
	}
}

func TestCloudflareAccessTransportUsesCloudflaredToken(t *testing.T) {
	token := testJWT(time.Now().Add(time.Hour))
	cloudflaredPath := filepath.Join(t.TempDir(), "cloudflared")
	if err := os.WriteFile(cloudflaredPath, []byte("#!/bin/sh\nprintf '%s\\n' '"+token+"'\n"), 0o700); err != nil {
		t.Fatalf("write fake cloudflared: %v", err)
	}

	transport := cloudflareAccessTransport{
		base: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if got := r.Header.Get("cf-access-token"); got != token {
				t.Fatalf("cf-access-token = %q", got)
			}
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		}),
		config: Config{},
		tokenProvider: newCloudflareTokenProvider(Config{
			CloudflareAccessAppURL: "https://logchef.example.test",
			CloudflaredPath:        cloudflaredPath,
		}),
	}

	req, err := http.NewRequest(http.MethodGet, "https://logchef.example.test/api/v1/me", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if _, err := transport.RoundTrip(req); err != nil {
		t.Fatalf("round trip failed: %v", err)
	}
}

func TestCustomCACertFile(t *testing.T) {
	certFile, err := os.CreateTemp(t.TempDir(), "ca-*.pem")
	if err != nil {
		t.Fatalf("create cert file: %v", err)
	}
	if _, err := certFile.WriteString(testCACertPEM); err != nil {
		t.Fatalf("write cert file: %v", err)
	}
	if err := certFile.Close(); err != nil {
		t.Fatalf("close cert file: %v", err)
	}

	_ = newHTTPClient(Config{CACertFile: certFile.Name()})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func testJWT(expiresAt time.Time) string {
	header := "eyJhbGciOiJub25lIn0"
	payload := base64RawURLEncode([]byte(fmt.Sprintf(`{"exp":%d}`, expiresAt.Unix())))
	return header + "." + payload + "."
}

func base64RawURLEncode(input []byte) string {
	encoding := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	var output []byte
	for i := 0; i < len(input); i += 3 {
		var chunk uint32
		remaining := len(input) - i
		chunk |= uint32(input[i]) << 16
		if remaining > 1 {
			chunk |= uint32(input[i+1]) << 8
		}
		if remaining > 2 {
			chunk |= uint32(input[i+2])
		}
		output = append(output, encoding[(chunk>>18)&0x3f], encoding[(chunk>>12)&0x3f])
		if remaining > 1 {
			output = append(output, encoding[(chunk>>6)&0x3f])
		}
		if remaining > 2 {
			output = append(output, encoding[chunk&0x3f])
		}
	}
	return string(output)
}

const testCACertPEM = `-----BEGIN CERTIFICATE-----
MIIDHzCCAsWgAwIBAgIUSdq7NH53pEHp6fcnOxitCXMDXMEwCgYIKoZIzj0EAwIw
gcAxCzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpDYWxpZm9ybmlhMRYwFAYDVQQHEw1T
YW4gRnJhbmNpc2NvMRkwFwYDVQQKExBDbG91ZGZsYXJlLCBJbmMuMRswGQYDVQQL
ExJ3d3cuY2xvdWRmbGFyZS5jb20xTDBKBgNVBAMTQ0dhdGV3YXkgQ0EgLSBDbG91
ZGZsYXJlIE1hbmFnZWQgRzIgNjZmMDUwODgzM2RjNzE0MTBkMzFjNGZkNzA1ODNj
YzMwIBcNMjUxMDA3MTAzMzAwWhgPMjA1NTA5MzAxMDMzMDBaMIHAMQswCQYDVQQG
EwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMNU2FuIEZyYW5jaXNj
bzEZMBcGA1UEChMQQ2xvdWRmbGFyZSwgSW5jLjEbMBkGA1UECxMSd3d3LmNsb3Vk
ZmxhcmUuY29tMUwwSgYDVQQDE0NHYXRld2F5IENBIC0gQ2xvdWRmbGFyZSBNYW5h
Z2VkIEcyIDY2ZjA1MDg4MzNkYzcxNDEwZDMxYzRmZDcwNTgzY2MzMFkwEwYHKoZI
zj0CAQYIKoZIzj0DAQcDQgAEfDdSLPhqrTFiTOy3IwPZlgGRVNY11Z3pQabE4jE3
dAgKTuTeEOlhyWHcDpsMU4KdMAUQM3qjuLaVsTtSTHctlaOBmDCBlTAOBgNVHQ8B
Af8EBAMCAQYwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUNijTiffnXguLBKou
gK1DANRBVJAwUwYDVR0fBEwwSjBIoEagRIZCaHR0cDovL2NybC5jbG91ZGZsYXJl
LmNvbS83ZWVkOTYyZi1hYzBiLTQ4YjktYWVmMS0xM2E4MDJmOWZkZTYuY3JsMAoG
CCqGSM49BAMCA0gAMEUCIDQJGpMMr02grReVQIlYypZ3bTux7zH7nG6M7blpDSzq
AiEA8imAFscCWxqxUWDp4x4i9RjtKlYk7Kk+sN6SH0B43JU=
-----END CERTIFICATE-----`
