---
name: search-with-exa
description: Use when you need to search the web or fetch webpage page content
---

# search-with-exa

Use this skill to search for information

## Workflow

Search with `exa-cli search` to get results.

You can search multiple times, or fetch page contents using `exa-cli fetch`.  Make sure that you have a good answer to the query before you stop searching.

## Finding `exa-cli`

If `exa-cli` is not immediately available:

1. Check whether `exa-cli` is installed with `command -v exa-cli`. 
2. You can also check if it exists in `scripts/`

If `exa-cli` is not available, you can ask the user to install it by running:

```sh
curl -fsSL https://raw.githubusercontent.com/alexhughson/exa-cli/main/install.sh | sh
```

Or else you can fetch the binary from:

https://github.com/alexhughson/exa-cli/releases/latest/download/exa-cli_<os>_<arch>.<ext>.

## Authentication

`exa-cli` will attempt to use saved authentication, or the Exa free tier, so you should be able to use it without authenticating.

If there are authentication failures, ask the user to run `<path_to_exa_cli> login` to add a token, and then you can retry.

## Usage Details

```sh
exa-cli --json search "query"
exa-cli --json fetch https://example.com
exa-cli --json advanced-search --type deep "query"
```

- `exa-cli search [--num-results N] "query"` for standard web search.
- `exa-cli fetch [--max-characters N] URL...` for page contents.
- `exa-cli advanced-search ... "query"` for filters, summaries, domains, crawl windows, and related Exa search options.
- `exa-cli login [api-key]` to save credentials when the free tier is not enough.
- `exa-cli logout` to remove saved credentials.

## Notes

- Use `--timeout` for long-running requests when needed.
- Keep the user's shell quoting intact for multi-word queries and repeated flags.
