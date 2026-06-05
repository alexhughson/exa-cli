# exa-cli

> ⚠️ This tool is largely vibe coded

A small CLI for Exa.ai

- Login with `exa-cli login`
- Search via `exa-cli search`
- Fetch web content via `exa-cli fetch`
- Exa advanced search via `exa-cli advanced-search`

This project was built because the Exa MCP server is unweildly to provision/authenticate/manage.  The CLI + Skill combination should be a bit more robust.

## Install

Install the latest release on macOS or Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/alexhughson/exa-cli/main/install.sh | sh
```

## Agent Skill

An agent skill bundle lives at `skills/search-with-exa`.  

## Authentication

The CLI will try Exa's free tier when no API key is configured. If the unauthenticated request is rejected or the free-tier limit is exhausted, it will tell you to run `exa-cli login`.

### API Key 

The CLI will use an `EXA_API_KEY` environment variable if it sees one:

```sh
EXA_API_KEY=your-key exa-cli search "latest AI search APIs"
```

### Saved Auth

Running `exa-cli login` will request an API key and persist them in your home directory.

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
- `EXA_CLI_CONFIG` - Where the config file with API keys is stored
- `EXA_API_BASE`
