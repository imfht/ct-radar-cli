import argparse
import os
import sys
import time

import requests

from . import __version__

DEFAULT_URL = "https://cert.imfht.com/api/search"


def main():
    parser = argparse.ArgumentParser(
        prog="ct-radar",
        description="CT Radar CLI — search Certificate Transparency logs for domains.\n\n"
                    "Anonymous: 1 request/min (no signup needed, perfect for ct-radar X.com | httpx).\n"
                    "Free signup at https://cert.imfht.com gets 100/day.\n"
                    "Pro $29/mo gets 10,000/day + Monitor + CSV export.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("query", help="Domain query (e.g. google.com)")
    parser.add_argument("--limit", type=int, default=100, help="Results limit (default 100, max 50 anonymous / 500 pro)")
    parser.add_argument("--offset", type=int, default=0, help="Offset for pagination")
    parser.add_argument("--key", help="API Key (or set CT_RADAR_KEY env). Optional — anonymous works at 1 req/min.")
    parser.add_argument(
        "--url",
        default=os.environ.get("CT_RADAR_URL", DEFAULT_URL),
        help=f"API URL (default: {DEFAULT_URL}, override with CT_RADAR_URL env)",
    )
    parser.add_argument("--exclude-expired", action="store_true",
                        help="Hide certificates whose validity period has ended")
    parser.add_argument("--exclude-wildcard", action="store_true",
                        help="Hide wildcard certificates (*.example.com)")
    parser.add_argument("--retry-on-throttle", action="store_true",
                        help="On 429 anonymous rate limit, sleep retry_after_seconds and retry once")
    parser.add_argument("--version", action="version", version=f"ct-radar {__version__}")

    args = parser.parse_args()

    headers = {"Accept": "application/json", "User-Agent": f"ct-radar-cli/{__version__}"}
    api_key = args.key or os.environ.get("CT_RADAR_KEY")
    if api_key:
        headers["X-API-Key"] = api_key

    params = {"q": args.query, "limit": args.limit, "offset": args.offset}
    if args.exclude_expired:
        params["exclude_expired"] = "true"
    if args.exclude_wildcard:
        params["exclude_wildcard"] = "true"

    response = _do_request(args.url, headers, params)

    # 429 = rate limit. 匿名用户最常见 (1/min)。--retry-on-throttle 自动等。
    if response.status_code == 429 and args.retry_on_throttle:
        try:
            data = response.json()
            wait = int(data.get("retry_after_seconds", 60))
        except Exception:
            wait = int(response.headers.get("Retry-After", "60"))
        print(f"# rate-limited, waiting {wait}s then retrying...", file=sys.stderr)
        time.sleep(wait + 1)
        response = _do_request(args.url, headers, params)

    if response.status_code == 429:
        try:
            data = response.json()
            msg = data.get("message") or data.get("error") or "rate limited"
            wait = data.get("retry_after_seconds")
        except Exception:
            msg = "rate limited"
            wait = response.headers.get("Retry-After")
        print(f"Error: {msg}", file=sys.stderr)
        if wait:
            print(f"  retry after {wait}s, or use --retry-on-throttle, or set CT_RADAR_KEY for higher quota.",
                  file=sys.stderr)
        sys.exit(2)

    if response.status_code == 401:
        print("Error: invalid API key (check CT_RADAR_KEY)", file=sys.stderr)
        sys.exit(1)

    if response.status_code != 200:
        print(f"Error: API returned {response.status_code}: {response.text[:200]}", file=sys.stderr)
        sys.exit(1)

    data = response.json()
    # Pipe-friendly: one unique domain per line, sorted. Plays with httpx/nuclei/etc.
    domains = {res.get("domain") for res in data.get("results", []) if res.get("domain")}
    for d in sorted(domains):
        print(d)


def _do_request(url: str, headers: dict, params: dict) -> requests.Response:
    try:
        return requests.get(url, headers=headers, params=params, timeout=30)
    except requests.RequestException as e:
        print(f"Error: network/HTTP failure: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
