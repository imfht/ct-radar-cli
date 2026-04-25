import argparse
import os
import sys

import requests

from . import __version__


def main():
    parser = argparse.ArgumentParser(
        prog="ct-radar",
        description="CT Radar CLI - Search Certificate Transparency logs for domains",
    )
    parser.add_argument("query", help="Domain query (e.g. google.com)")
    parser.add_argument("--limit", type=int, default=100, help="Results limit")
    parser.add_argument("--offset", type=int, default=0, help="Offset for pagination")
    parser.add_argument("--key", help="API Key (or set CT_RADAR_KEY env)")
    parser.add_argument(
        "--url",
        default=os.environ.get("CT_RADAR_URL", "https://api.ct-radar.com/api/search"),
        help="API URL (default: https://api.ct-radar.com/api/search)",
    )
    parser.add_argument("--version", action="version", version=f"ct-radar {__version__}")

    args = parser.parse_args()

    api_key = args.key or os.environ.get("CT_RADAR_KEY")
    if not api_key:
        print("Error: API Key required. Set via --key or CT_RADAR_KEY env.", file=sys.stderr)
        sys.exit(1)

    headers = {"X-API-Key": api_key, "Accept": "application/json"}
    params = {"q": args.query, "limit": args.limit, "offset": args.offset}

    try:
        response = requests.get(args.url, headers=headers, params=params, timeout=30)
    except requests.RequestException as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

    if response.status_code == 429:
        print("Error: Daily quota exceeded.", file=sys.stderr)
        sys.exit(1)
    if response.status_code != 200:
        print(f"Error: API returned {response.status_code}: {response.text}", file=sys.stderr)
        sys.exit(1)

    data = response.json()
    domains = {res.get("domain") for res in data.get("results", []) if res.get("domain")}
    for d in sorted(domains):
        print(d)


if __name__ == "__main__":
    main()
