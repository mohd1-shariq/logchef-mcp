# Codex Setup With Cloudflare Access

This guide is for Logchef deployments behind Cloudflare Access, such as a
company-internal Logchef URL that requires browser SSO or WARP.

The patched MCP server can get a Cloudflare Access JWT through `cloudflared`,
attach it to every Logchef API request, and refresh it when it expires. You do
not need to copy `CF_Authorization` cookies from the browser.

## What This Fork Adds

- Custom CA bundle support with `LOGCHEF_CA_CERT_FILE`
- Explicit Cloudflare Access service-token headers
- Static Cloudflare Access token support
- Browser-cookie fallback for emergency debugging
- `cloudflared access token` support for SSO-protected Access apps
- Optional `cloudflared access login` when the token is missing or expired

## Prerequisites

- Go 1.25 or newer
- A Logchef API token from Profile > API Tokens in the Logchef UI
- `cloudflared` installed locally
- Access to the Cloudflare Access application in your browser

On macOS:

```bash
brew install cloudflared
```

## Build The MCP Server

Use the patched fork or branch that contains this document:

```bash
git clone <patched-logchef-mcp-fork-url> logchef-mcp
cd logchef-mcp
go build -o ./logchef-mcp.bin ./cmd/logchef-mcp
```

Keep the binary path stable because Codex stores the absolute path in its MCP
configuration.

## One-Command Codex Setup

The repository includes a setup script that builds the binary, optionally runs
Cloudflare Access login, backs up `~/.codex/config.toml`, and writes the Codex
MCP config:

```bash
./scripts/setup-codex-logchef.sh
```

If `LOGCHEF_API_KEY` is not already set, the script prompts for it without
echoing the value.

For `https://logchef.cars24.team` and `https://logchef-dev.cars24.team`, the
script automatically enables the `cloudflared` Cloudflare Access flow. If you
are targeting a custom non-Access Logchef URL, pass `--no-cloudflare-access`.

With a Cloudflare Gateway/WARP CA file:

```bash
./scripts/setup-codex-logchef.sh \
  --ca-cert-file ~/.codex/logchef-cloudflare-gateway-ca.pem
```

For unattended Cloudflare Access service-token setup:

```bash
LOGCHEF_API_KEY=<logchef_api_key> ./scripts/setup-codex-logchef.sh \
  --service-token-client-id <client_id> \
  --service-token-client-secret <client_secret> \
  --no-login
```

Use `./scripts/setup-codex-logchef.sh --help` for all supported modes.

The script is safe to rerun. It replaces only the managed
`[mcp_servers.logchef]` and `[mcp_servers.logchef.env]` blocks, preserves the
rest of the Codex config, writes a timestamped backup, and redacts secrets in
dry-run output.

## Log In To Cloudflare Access

Run:

```bash
cloudflared access login https://logchef.cars24.team
```

This opens the browser for SSO and stores the Access token in the normal
`cloudflared` token cache.

You can verify that `cloudflared` can return a token without printing it:

```bash
cloudflared access token -app=https://logchef.cars24.team | wc -c
```

## Configure Codex

Edit `~/.codex/config.toml`:

```toml
[mcp_servers.logchef]
command = "/absolute/path/to/logchef-mcp/logchef-mcp.bin"
args = ["-t", "stdio", "--disable-admin"]

[mcp_servers.logchef.env]
LOGCHEF_URL = "https://logchef.cars24.team"
LOGCHEF_API_KEY = "<logchef_api_key>"
LOGCHEF_CF_ACCESS_APP_URL = "https://logchef.cars24.team"
LOGCHEF_CLOUDFLARED_PATH = "/opt/homebrew/bin/cloudflared"
LOGCHEF_CF_ACCESS_AUTO_LOGIN = "true"
```

Use the correct `cloudflared` path for the machine:

```bash
which cloudflared
```

Restart Codex after changing MCP config. Codex starts stdio MCP servers when the
session starts, so an already-open session will keep using the old process.

## Optional WARP CA Configuration

If requests fail with an error like `x509: certificate signed by unknown
authority`, export or save the Cloudflare Gateway root certificate as a PEM file
and add:

```toml
LOGCHEF_CA_CERT_FILE = "/absolute/path/to/cloudflare-gateway-ca.pem"
```

Prefer `LOGCHEF_CA_CERT_FILE` over `LOGCHEF_INSECURE_SKIP_VERIFY=true`.

## Verify The MCP Server

Check that Codex sees the server:

```bash
codex mcp get logchef
```

Expected shape:

```text
logchef
  enabled: true
  transport: stdio
  command: /absolute/path/to/logchef-mcp/logchef-mcp.bin
  args: -t stdio --disable-admin
```

Then reopen Codex and ask it to list Logchef teams, sources, or service names.

## Token Lifetime

The Cloudflare Access JWT lifetime is controlled by the Cloudflare Access
application policy. In the CARS24 setup observed during testing, the token was
valid for 24 hours. With `LOGCHEF_CF_ACCESS_AUTO_LOGIN=true`, the MCP server
will run `cloudflared access login` when `cloudflared access token` cannot
return a usable token.

## Other Cloudflare Access Modes

For non-SSO automation, the server also supports Cloudflare Access service
tokens:

```toml
LOGCHEF_CF_ACCESS_CLIENT_ID = "<client_id>"
LOGCHEF_CF_ACCESS_CLIENT_SECRET = "<client_secret>"
```

For temporary debugging only, it can also pass browser cookie values:

```toml
LOGCHEF_CF_AUTHORIZATION = "<CF_Authorization JWT>"
LOGCHEF_CF_APPSESSION = "<CF_AppSession>"
```

Do not use copied browser cookies as the normal setup path.
