# Logchef MCP Server Setup

Connect your AI assistant to [Logchef](https://logchef.app) for natural language log exploration and analysis powered by ClickHouse.

## Prerequisites

- A running Logchef instance
- A Logchef API token (generate one from Profile > API Tokens in the Logchef UI)

## Quick Start

Choose your AI tool below to get started.

---

### Claude Code

```bash
claude mcp add logchef -- logchef-mcp
```

Set your credentials:

```bash
export LOGCHEF_URL=https://your-logchef-instance.com
export LOGCHEF_API_KEY=your_api_token
```

Verify it works:

```
/mcp
```

You should see `logchef` listed with its tools.

### Claude Desktop

Add to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "logchef": {
      "command": "logchef-mcp",
      "env": {
        "LOGCHEF_URL": "https://your-logchef-instance.com",
        "LOGCHEF_API_KEY": "your_api_token"
      }
    }
  }
}
```

> If you see `Error: spawn logchef-mcp ENOENT`, use the full path to the binary.

### Cursor

Open Cursor Settings > MCP Servers, then add:

```json
{
  "mcpServers": {
    "logchef": {
      "command": "logchef-mcp",
      "env": {
        "LOGCHEF_URL": "https://your-logchef-instance.com",
        "LOGCHEF_API_KEY": "your_api_token"
      }
    }
  }
}
```

### VS Code (Copilot)

Add to `.vscode/settings.json`:

```json
{
  "mcp": {
    "servers": {
      "logchef": {
        "command": "logchef-mcp",
        "env": {
          "LOGCHEF_URL": "https://your-logchef-instance.com",
          "LOGCHEF_API_KEY": "your_api_token"
        }
      }
    }
  }
}
```

For a remote MCP server (streamable HTTP mode):

```json
{
  "mcp": {
    "servers": {
      "logchef": {
        "type": "http",
        "url": "http://localhost:8000/mcp",
        "headers": {
          "X-Logchef-URL": "https://your-logchef-instance.com",
          "X-Logchef-API-Key": "your_api_token"
        }
      }
    }
  }
}
```

### Codex CLI

```bash
# Add MCP server
codex mcp add logchef -- logchef-mcp

# Or with explicit env vars
LOGCHEF_URL=https://your-logchef-instance.com \
LOGCHEF_API_KEY=your_api_token \
codex mcp add logchef -- logchef-mcp
```

For Logchef instances protected by Cloudflare Access, use the dedicated
[Codex + Cloudflare Access setup guide](codex-cloudflare-access.md).

### Windsurf

Open Windsurf Settings > MCP, then add:

```json
{
  "mcpServers": {
    "logchef": {
      "command": "logchef-mcp",
      "env": {
        "LOGCHEF_URL": "https://your-logchef-instance.com",
        "LOGCHEF_API_KEY": "your_api_token"
      }
    }
  }
}
```

### Docker

For any tool that supports MCP via stdio:

```json
{
  "mcpServers": {
    "logchef": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-e", "LOGCHEF_URL",
        "-e", "LOGCHEF_API_KEY",
        "ghcr.io/mr-karan/logchef-mcp:latest",
        "-t", "stdio"
      ],
      "env": {
        "LOGCHEF_URL": "https://your-logchef-instance.com",
        "LOGCHEF_API_KEY": "your_api_token"
      }
    }
  }
}
```

---

## Installation

### Binary (recommended)

Download the latest release from the [releases page](https://github.com/mr-karan/logchef-mcp/releases) and place it in your `$PATH`.

### Go install

```bash
go install github.com/mr-karan/logchef-mcp/cmd/logchef-mcp@latest
```

### Build from source

```bash
git clone https://github.com/mr-karan/logchef-mcp.git
cd logchef-mcp
go build -o logchef-mcp ./cmd/logchef-mcp
```

### Docker

```bash
docker pull ghcr.io/mr-karan/logchef-mcp:latest
```

---

## Transport Modes

The server supports three transport modes:

| Mode | Flag | Use case |
|------|------|----------|
| **stdio** (default) | `-t stdio` | Direct integration with AI assistants |
| **SSE** | `-t sse` | Legacy web-based clients |
| **Streamable HTTP** | `-t streamable-http` | Multi-client HTTP deployments |

For HTTP modes, the server listens on `localhost:8000` by default. Override with `--address`.

---

## Authentication

### Environment Variables (stdio mode)

```bash
export LOGCHEF_URL=https://your-logchef-instance.com
export LOGCHEF_API_KEY=your_api_token
```

### HTTP Headers (SSE / Streamable HTTP mode)

When running in HTTP mode, clients can pass credentials via headers:

- `X-Logchef-URL` — Logchef instance URL
- `X-Logchef-API-Key` — API token

Headers take precedence over environment variables. If headers are absent, the server falls back to env vars.

### Cloudflare Access / private CA support

For Logchef instances protected by Cloudflare Access, configure one of the following:

```bash
# Preferred interactive user flow. Run this once, or set auto-login below.
cloudflared access login https://your-logchef-instance.com
export LOGCHEF_CF_ACCESS_APP_URL=https://your-logchef-instance.com

# Optional: let the MCP server trigger browser login if token retrieval fails.
export LOGCHEF_CF_ACCESS_AUTO_LOGIN=true

# Preferred for automation.
export LOGCHEF_CF_ACCESS_CLIENT_ID=your_service_token_client_id
export LOGCHEF_CF_ACCESS_CLIENT_SECRET=your_service_token_client_secret

# Or use an existing Access JWT.
export LOGCHEF_CF_ACCESS_TOKEN=your_cf_access_jwt

# Temporary browser-session fallback.
export LOGCHEF_CF_AUTHORIZATION=your_cf_authorization_cookie_value
export LOGCHEF_CF_APPSESSION=your_cf_appsession_cookie_value
```

For private TLS roots, such as Cloudflare Gateway/WARP inspection CAs:

```bash
export LOGCHEF_CA_CERT_FILE=/path/to/company-root-ca.pem

# Last-resort local workaround only.
export LOGCHEF_INSECURE_SKIP_VERIFY=true
```

The same values can be passed as headers in SSE / Streamable HTTP mode:

- `X-Logchef-CF-Access-Client-ID`
- `X-Logchef-CF-Access-Client-Secret`
- `X-Logchef-CF-Access-Token`
- `X-Logchef-CF-Access-App-URL`
- `X-Logchef-Cloudflared-Path`
- `X-Logchef-CF-Access-Auto-Login`
- `X-Logchef-CF-Authorization`
- `X-Logchef-CF-AppSession`
- `X-Logchef-CA-Cert-File`
- `X-Logchef-Insecure-Skip-Verify`

For unattended MCP usage, prefer Cloudflare Access service tokens. Browser-session cookies are short-lived and should only be used for temporary local testing.

---

## Tool Configuration

Selectively enable or disable tool categories:

```bash
# Only enable log querying and profile tools
logchef-mcp --enabled-tools profile,sources,logs,logchefql

# Disable admin tools
logchef-mcp --disable-admin

# Disable telemetry tools
logchef-mcp --disable-telemetry
```

Available categories: `profile`, `sources`, `logs`, `logchefql`, `investigate`, `admin`, `analysis`, `telemetry`, `discover`

---

## Debug Mode

Enable verbose HTTP logging between the MCP server and Logchef API:

```bash
logchef-mcp -debug
```

In Claude Desktop config:

```json
{
  "mcpServers": {
    "logchef": {
      "command": "logchef-mcp",
      "args": ["-debug"],
      "env": {
        "LOGCHEF_URL": "https://your-logchef-instance.com",
        "LOGCHEF_API_KEY": "your_api_token"
      }
    }
  }
}
```
