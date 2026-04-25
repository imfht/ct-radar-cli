# ct-radar (Go CLI)

Go CLI for CT Radar - search Certificate Transparency logs for domains.

## Install

```bash
# Via Go
go install github.com/imfht/ct-radar-cli/cli@latest

# Or download a prebuilt binary from GitHub Releases
# https://github.com/imfht/ct-radar-cli/releases
```

## Usage

```bash
export CT_RADAR_KEY=your_api_key
ct-radar example.com 200
```

Prints one unique domain per line, sorted. Pipe into `httpx`, `nuclei`, etc.
