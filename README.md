# ct-radar-cli

CLI clients for [CT Radar](https://ct-radar.com) — search Certificate Transparency logs for domains.

Two implementations live in this repo:

- [`cli/`](./cli) — Go CLI, distributed as static binaries via GitHub Releases (`go install` works too)
- [`python-cli/`](./python-cli) — Python CLI, distributed via PyPI (`pip install ct-radar`)

Both clients speak the same HTTP API and produce identical pipe-friendly output (one unique domain per line, sorted) so they slot into recon pipelines with `httpx` / `nuclei` / etc.

## Quick start

```bash
export CT_RADAR_KEY=your_api_key

# Python
pip install ct-radar
ct-radar example.com

# Go
go install github.com/imfht/ct-radar-cli/cli@latest
ct-radar example.com
```

## Releasing

- Go binaries: tag `cli-vX.Y.Z` → goreleaser publishes to GitHub Releases
- Python wheel: tag `pycli-vX.Y.Z` → publishes to PyPI via OIDC trusted publishing
