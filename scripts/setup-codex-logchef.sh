#!/usr/bin/env bash
set -euo pipefail

script_name="$(basename "$0")"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)"
repo_root="$(cd "$script_dir/.." && pwd -P)"

server_name="logchef"
logchef_url="${LOGCHEF_URL:-https://logchef.cars24.team}"
api_key="${LOGCHEF_API_KEY:-}"
config_path="${CODEX_CONFIG:-$HOME/.codex/config.toml}"
binary_path="$repo_root/logchef-mcp.bin"
build_binary=1
disable_admin=1
dry_run=0
non_interactive=0
install_cloudflared=0
verify_codex=1

cloudflare_access=0
cloudflare_app_url="${LOGCHEF_CF_ACCESS_APP_URL:-}"
cloudflared_path="${LOGCHEF_CLOUDFLARED_PATH:-}"
cloudflare_auto_login="${LOGCHEF_CF_ACCESS_AUTO_LOGIN:-true}"
run_login="auto"

ca_cert_file="${LOGCHEF_CA_CERT_FILE:-}"
insecure_skip_verify="${LOGCHEF_INSECURE_SKIP_VERIFY:-}"
cf_client_id="${LOGCHEF_CF_ACCESS_CLIENT_ID:-}"
cf_client_secret="${LOGCHEF_CF_ACCESS_CLIENT_SECRET:-}"
cf_access_token="${LOGCHEF_CF_ACCESS_TOKEN:-}"
cf_authorization="${LOGCHEF_CF_AUTHORIZATION:-}"
cf_appsession="${LOGCHEF_CF_APPSESSION:-}"

if [[ -n "$cloudflare_app_url" ]]; then
  cloudflare_access=1
fi

usage() {
  cat <<USAGE
Usage:
  $script_name [options]

Common setup:
  $script_name --cloudflare-access

Options:
  --url <url>                         Logchef URL. Default: $logchef_url
  --api-key <key>                     Logchef API key. Can also use LOGCHEF_API_KEY.
  --config <path>                     Codex config path. Default: ~/.codex/config.toml
  --name <server-name>                MCP server name. Default: logchef
  --binary <path>                     Output/path for MCP binary. Default: ./logchef-mcp.bin
  --build                             Build the binary before writing config. Default.
  --no-build                          Do not build; require the binary to already exist.
  --disable-admin                     Add --disable-admin to the MCP args. Default.
  --enable-admin                      Do not add --disable-admin.
  --dry-run                           Validate and print a redacted config preview only.
  --no-verify                         Skip the final "codex mcp get" verification.
  --non-interactive                   Never prompt for missing values.

Cloudflare Access SSO:
  --cloudflare-access                 Use cloudflared Access token flow.
  --access-app-url <url>              Cloudflare Access app URL. Defaults to --url.
  --cloudflared-path <path>           cloudflared path. Defaults to command lookup.
  --login                             Run cloudflared access login. Default with --cloudflare-access.
  --no-login                          Skip browser login.
  --auto-login                        Let MCP trigger cloudflared login if token is expired. Default.
  --no-auto-login                     Do not let MCP trigger login.
  --install-cloudflared               Install cloudflared with Homebrew if missing.

Cloudflare Access service/non-interactive modes:
  --service-token-client-id <id>      CF Access service token client id.
  --service-token-client-secret <sec> CF Access service token client secret.
  --access-token <jwt>                Static Cloudflare Access JWT.
  --cf-authorization <jwt>            Temporary CF_Authorization cookie value.
  --cf-appsession <value>             Temporary CF_AppSession cookie value.

TLS:
  --ca-cert-file <path>               Extra PEM root CA bundle, useful for WARP/Gateway.
  --insecure-skip-verify             Last-resort local workaround for private TLS issues.

Examples:
  $script_name --cloudflare-access
  $script_name --cloudflare-access --ca-cert-file ~/.codex/logchef-cloudflare-gateway-ca.pem
  LOGCHEF_API_KEY=logchef_xxx $script_name --service-token-client-id xxx --service-token-client-secret yyy --no-login
USAGE
}

die() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

warn() {
  printf 'warning: %s\n' "$*" >&2
}

info() {
  printf '%s\n' "$*"
}

need_value() {
  local flag="$1"
  local value="${2-}"
  if [[ -z "$value" ]]; then
    die "$flag requires a value"
  fi
}

expand_path() {
  local path="$1"
  case "$path" in
    "~") printf '%s\n' "$HOME" ;;
    "~/"*) printf '%s/%s\n' "$HOME" "${path#~/}" ;;
    *) printf '%s\n' "$path" ;;
  esac
}

absolute_path() {
  local path dir base
  local create_parent="${2:-0}"
  path="$(expand_path "$1")"
  dir="$(dirname "$path")"
  base="$(basename "$path")"
  if [[ "$create_parent" -eq 1 ]]; then
    mkdir -p "$dir"
  elif [[ ! -d "$dir" ]]; then
    die "parent directory does not exist: $dir"
  fi
  dir="$(cd "$dir" && pwd -P)"
  printf '%s/%s\n' "$dir" "$base"
}

toml_escape() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  printf '%s' "$value"
}

parse_bool() {
  case "$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')" in
    1|true|yes|y|on) printf 'true' ;;
    *) printf 'false' ;;
  esac
}

while (($#)); do
  case "$1" in
    --url)
      need_value "$1" "${2-}"
      logchef_url="$2"
      shift 2
      ;;
    --api-key)
      need_value "$1" "${2-}"
      api_key="$2"
      shift 2
      ;;
    --config)
      need_value "$1" "${2-}"
      config_path="$2"
      shift 2
      ;;
    --name)
      need_value "$1" "${2-}"
      server_name="$2"
      shift 2
      ;;
    --binary)
      need_value "$1" "${2-}"
      binary_path="$2"
      shift 2
      ;;
    --build)
      build_binary=1
      shift
      ;;
    --no-build)
      build_binary=0
      shift
      ;;
    --disable-admin)
      disable_admin=1
      shift
      ;;
    --enable-admin)
      disable_admin=0
      shift
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    --no-verify)
      verify_codex=0
      shift
      ;;
    --non-interactive)
      non_interactive=1
      shift
      ;;
    --cloudflare-access)
      cloudflare_access=1
      shift
      ;;
    --access-app-url)
      need_value "$1" "${2-}"
      cloudflare_access=1
      cloudflare_app_url="$2"
      shift 2
      ;;
    --cloudflared-path)
      need_value "$1" "${2-}"
      cloudflare_access=1
      cloudflared_path="$2"
      shift 2
      ;;
    --login)
      run_login=1
      shift
      ;;
    --no-login)
      run_login=0
      shift
      ;;
    --auto-login)
      cloudflare_auto_login=true
      shift
      ;;
    --no-auto-login)
      cloudflare_auto_login=false
      shift
      ;;
    --install-cloudflared)
      install_cloudflared=1
      cloudflare_access=1
      shift
      ;;
    --service-token-client-id)
      need_value "$1" "${2-}"
      cf_client_id="$2"
      shift 2
      ;;
    --service-token-client-secret)
      need_value "$1" "${2-}"
      cf_client_secret="$2"
      shift 2
      ;;
    --access-token)
      need_value "$1" "${2-}"
      cf_access_token="$2"
      shift 2
      ;;
    --cf-authorization)
      need_value "$1" "${2-}"
      cf_authorization="$2"
      shift 2
      ;;
    --cf-appsession)
      need_value "$1" "${2-}"
      cf_appsession="$2"
      shift 2
      ;;
    --ca-cert-file)
      need_value "$1" "${2-}"
      ca_cert_file="$2"
      shift 2
      ;;
    --insecure-skip-verify)
      insecure_skip_verify=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "unknown option: $1"
      ;;
  esac
done

case "$server_name" in
  *[!A-Za-z0-9_-]*|'') die "--name must contain only letters, numbers, underscore, or hyphen" ;;
esac

if [[ "$logchef_url" != http://* && "$logchef_url" != https://* ]]; then
  die "--url must start with http:// or https://"
fi
logchef_url="${logchef_url%/}"

if [[ -z "$api_key" && "$non_interactive" -eq 0 && -t 0 ]]; then
  read -r -s -p "Logchef API key: " api_key
  printf '\n'
fi
if [[ -z "$api_key" ]]; then
  die "missing Logchef API key; pass --api-key or set LOGCHEF_API_KEY"
fi

config_path="$(absolute_path "$config_path" 1)"
binary_path="$(absolute_path "$binary_path" 1)"

if [[ -n "$ca_cert_file" ]]; then
  ca_cert_file="$(absolute_path "$ca_cert_file" 0)"
  [[ -r "$ca_cert_file" ]] || die "CA cert file is not readable: $ca_cert_file"
fi

insecure_skip_verify="$(parse_bool "$insecure_skip_verify")"
cloudflare_auto_login="$(parse_bool "$cloudflare_auto_login")"

if [[ -n "$cf_client_id" || -n "$cf_client_secret" ]]; then
  [[ -n "$cf_client_id" && -n "$cf_client_secret" ]] || die "service token mode requires both client id and client secret"
fi

cloudflare_modes=0
[[ "$cloudflare_access" -eq 1 ]] && cloudflare_modes=$((cloudflare_modes + 1))
[[ -n "$cf_client_id" ]] && cloudflare_modes=$((cloudflare_modes + 1))
[[ -n "$cf_access_token" ]] && cloudflare_modes=$((cloudflare_modes + 1))
[[ -n "$cf_authorization" || -n "$cf_appsession" ]] && cloudflare_modes=$((cloudflare_modes + 1))

if [[ "$cloudflare_modes" -gt 1 ]]; then
  die "choose only one Cloudflare Access mode: cloudflared SSO, service token, static token, or browser cookies"
fi

if [[ "$cloudflare_access" -eq 1 ]]; then
  if [[ -z "$cloudflare_app_url" ]]; then
    cloudflare_app_url="$logchef_url"
  fi
  cloudflare_app_url="${cloudflare_app_url%/}"

  if [[ -z "$cloudflared_path" ]]; then
    if command -v cloudflared >/dev/null 2>&1; then
      cloudflared_path="$(command -v cloudflared)"
    elif [[ "$install_cloudflared" -eq 1 ]]; then
      if [[ "$dry_run" -eq 0 ]]; then
        command -v brew >/dev/null 2>&1 || die "Homebrew is required for --install-cloudflared on macOS"
        brew install cloudflared
        cloudflared_path="$(command -v cloudflared || true)"
      else
        cloudflared_path="cloudflared"
      fi
    elif [[ "$dry_run" -eq 1 ]]; then
      cloudflared_path="cloudflared"
    else
      die "cloudflared not found; install it or pass --cloudflared-path"
    fi
  fi
  if [[ "$dry_run" -eq 0 ]]; then
    if [[ "$cloudflared_path" != */* ]]; then
      cloudflared_path="$(command -v "$cloudflared_path" || true)"
      [[ -n "$cloudflared_path" ]] || die "cloudflared not found in PATH"
    fi
    cloudflared_path="$(absolute_path "$cloudflared_path" 0)"
    [[ -x "$cloudflared_path" ]] || die "cloudflared is not executable: $cloudflared_path"
  elif [[ "$cloudflared_path" == */* && -d "$(dirname "$cloudflared_path")" ]]; then
    cloudflared_path="$(absolute_path "$cloudflared_path" 0)"
  fi

  if [[ "$run_login" == "auto" ]]; then
    run_login=1
  fi
else
  run_login=0
fi

if [[ -n "$cf_authorization" || -n "$cf_appsession" ]]; then
  warn "browser cookie mode is intended only for temporary local debugging"
fi

if [[ "$insecure_skip_verify" == "true" ]]; then
  warn "LOGCHEF_INSECURE_SKIP_VERIFY disables TLS verification; prefer --ca-cert-file"
fi

if [[ "$dry_run" -eq 0 && "$build_binary" -eq 1 ]]; then
  command -v go >/dev/null 2>&1 || die "go is required to build the MCP binary"
  info "Building $binary_path"
  (cd "$repo_root" && go build -o "$binary_path" ./cmd/logchef-mcp)
fi

if [[ "$dry_run" -eq 0 && ! -x "$binary_path" ]]; then
  die "MCP binary is missing or not executable: $binary_path"
elif [[ "$dry_run" -eq 1 && ! -e "$binary_path" ]]; then
  warn "dry run: MCP binary does not exist yet and would be built at $binary_path"
fi

if [[ "$dry_run" -eq 0 && "$cloudflare_access" -eq 1 && "$run_login" -eq 1 ]]; then
  info "Opening Cloudflare Access login for $cloudflare_app_url"
  "$cloudflared_path" access login "$cloudflare_app_url"
fi

emit_env_line() {
  local key="$1"
  local value="$2"
  local sensitive="${3:-0}"
  local redacted="${4:-0}"
  [[ -n "$value" ]] || return 0
  if [[ "$redacted" -eq 1 && "$sensitive" -eq 1 ]]; then
    value="<redacted>"
  fi
  printf '%s = "%s"\n' "$key" "$(toml_escape "$value")"
}

emit_config_block() {
  local redacted="${1:-0}"
  printf '[mcp_servers.%s]\n' "$server_name"
  printf 'command = "%s"\n' "$(toml_escape "$binary_path")"
  if [[ "$disable_admin" -eq 1 ]]; then
    printf 'args = ["-t", "stdio", "--disable-admin"]\n'
  else
    printf 'args = ["-t", "stdio"]\n'
  fi
  printf '\n'
  printf '[mcp_servers.%s.env]\n' "$server_name"
  emit_env_line "LOGCHEF_URL" "$logchef_url" 0 "$redacted"
  emit_env_line "LOGCHEF_API_KEY" "$api_key" 1 "$redacted"
  emit_env_line "LOGCHEF_CA_CERT_FILE" "$ca_cert_file" 0 "$redacted"
  if [[ "$insecure_skip_verify" == "true" ]]; then
    emit_env_line "LOGCHEF_INSECURE_SKIP_VERIFY" "true" 0 "$redacted"
  fi
  if [[ "$cloudflare_access" -eq 1 ]]; then
    emit_env_line "LOGCHEF_CF_ACCESS_APP_URL" "$cloudflare_app_url" 0 "$redacted"
    emit_env_line "LOGCHEF_CLOUDFLARED_PATH" "$cloudflared_path" 0 "$redacted"
    emit_env_line "LOGCHEF_CF_ACCESS_AUTO_LOGIN" "$cloudflare_auto_login" 0 "$redacted"
  fi
  emit_env_line "LOGCHEF_CF_ACCESS_CLIENT_ID" "$cf_client_id" 1 "$redacted"
  emit_env_line "LOGCHEF_CF_ACCESS_CLIENT_SECRET" "$cf_client_secret" 1 "$redacted"
  emit_env_line "LOGCHEF_CF_ACCESS_TOKEN" "$cf_access_token" 1 "$redacted"
  emit_env_line "LOGCHEF_CF_AUTHORIZATION" "$cf_authorization" 1 "$redacted"
  emit_env_line "LOGCHEF_CF_APPSESSION" "$cf_appsession" 1 "$redacted"
}

if [[ "$dry_run" -eq 1 ]]; then
  info "Dry run: would write this redacted Codex MCP config to $config_path"
  printf '\n'
  emit_config_block 1
  exit 0
fi

mkdir -p "$(dirname "$config_path")"
if [[ ! -f "$config_path" ]]; then
  : > "$config_path"
  chmod 600 "$config_path"
fi

timestamp="$(date +%Y%m%d%H%M%S)"
backup_path="$config_path.bak.$timestamp"
cp -p "$config_path" "$backup_path"

tmp_existing="$(mktemp "${config_path}.tmp.XXXXXX")"
tmp_new="$(mktemp "${config_path}.new.XXXXXX")"

awk -v name="$server_name" '
  $0 == "[mcp_servers." name "]" || $0 == "[mcp_servers." name ".env]" {
    skip = 1
    next
  }
  /^\[/ {
    skip = 0
  }
  !skip {
    print
  }
' "$config_path" > "$tmp_existing"

{
  cat "$tmp_existing"
  printf '\n'
  emit_config_block 0
} > "$tmp_new"

mv "$tmp_new" "$config_path"
rm -f "$tmp_existing"
chmod 600 "$config_path"

info "Updated Codex MCP config: $config_path"
info "Backup written to: $backup_path"

if [[ "$verify_codex" -eq 1 ]] && command -v codex >/dev/null 2>&1; then
  info "Codex now sees:"
  codex mcp get "$server_name" || true
elif [[ "$verify_codex" -eq 1 ]]; then
  warn "codex CLI not found in PATH; restart Codex and verify the MCP server from the app"
fi

info "Restart Codex so it starts the updated $server_name MCP process."
