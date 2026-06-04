# exa-cli

A small Go CLI for the primary Exa MCP tools:

- `web_search_exa` via `exa-cli search`
- `web_fetch_exa` via `exa-cli fetch`
- `web_search_advanced_exa` via `exa-cli advanced-search`

## Build

```sh
go build -o dist/exa-cli ./cmd/exa-cli
```

The result is a single binary.

## Authentication

The CLI prefers `EXA_API_KEY`:

```sh
EXA_API_KEY=your-key exa-cli search "latest AI search APIs"
```

You can also save credentials in `~/.exa-cli/config.json`:

```sh
exa-cli auth your-key
exa-cli auth status
exa-cli auth logout
```

The config directory is created with `0700` permissions and the config file is written with `0600` permissions.

## Usage

```sh
exa-cli search --num-results 5 "agent benchmarks"
exa-cli fetch --max-characters 3000 https://docs.exa.ai
exa-cli advanced-search --type deep --category "research paper" --include-domain arxiv.org --summary "LLM agents"
```

Use `--json` before the command for raw API JSON:

```sh
exa-cli --json search "agent benchmarks"
```

Global flags:

- `--json`
- `--timeout 60s`
- `--api-base https://api.exa.ai`

Environment variables:

- `EXA_API_KEY`
- `EXA_CLI_CONFIG`
- `EXA_API_BASE`
