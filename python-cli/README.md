# ct-radar

Python CLI for CT Radar - search Certificate Transparency logs for domains.

## Install

```bash
pip install ct-radar
```

## Usage

```bash
export CT_RADAR_KEY=your_api_key
ct-radar example.com --limit 200
```

Pipe-friendly: prints one unique domain per line, sorted. Plays well with `httpx`, `nuclei`, etc.

```bash
ct-radar example.com | httpx -silent
```
