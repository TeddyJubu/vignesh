#!/usr/bin/env python3
"""SOUL: prefer Scrapling MCP for structured web scraping."""
from __future__ import annotations

from pathlib import Path

SOUL = Path("/root/.hermes/SOUL.md")
MARKER = "## WEB SCRAPING — Scrapling (system instruction)"
ANCHOR = "## AUTONOMOUS AGENT — owners (system instruction)"

BLOCK = """
## WEB SCRAPING — Scrapling (system instruction)

**Installed on this host.** Hermes exposes Scrapling via MCP (`mcp_scrapling_*` tools).

### When to use Scrapling (preferred)
- Lead lists, business directories, review pages, Maps-style listings
- Need **CSS-targeted** extraction (less token waste than dumping full HTML)
- Anti-bot / dynamic pages: `stealthy_fetch`, `fetch`, `bulk_get`
- Screenshots of pages for verification

### Tool choice
| Need | Tool |
|------|------|
| Simple static page | `get` / `bulk_get` |
| JS-rendered page | `fetch` / `bulk_fetch` |
| Cloudflare / heavy protection | `stealthy_fetch` |
| Multi-page same site | `open_session` + repeated fetches, then `close_session` |

### Workflow
1. Use Scrapling MCP tools (not raw `urllib` or Composio workbench Python) for scrape tasks.
2. Narrow with CSS selectors before passing content onward.
3. Save tabular results to `~/.hermes/reports/*.csv`; summarize in chat + `MEDIA:` for owners.

### Do not
- Scrape Google Search results HTML via ad-hoc Python (blocked/unreliable) — use Scrapling on target sites or proper search APIs.
- Paste megabytes of HTML into WhatsApp.

Docs: https://github.com/D4Vinci/Scrapling · https://scrapling.readthedocs.io/en/latest/ai/mcp-server.html

"""


def main() -> None:
    text = SOUL.read_text(encoding="utf-8")
    if MARKER in text:
        print("ok: Scrapling SOUL section already present")
        return
    if ANCHOR in text:
        text = text.replace(ANCHOR, BLOCK.strip() + "\n\n" + ANCHOR, 1)
    else:
        anchor = "## WHATSAPP — production stack (system instruction)"
        if anchor not in text:
            raise SystemExit("SOUL anchor not found")
        text = text.replace(anchor, BLOCK.strip() + "\n\n" + anchor, 1)
    SOUL.write_text(text, encoding="utf-8")
    print("ok: SOUL Scrapling section added")


if __name__ == "__main__":
    main()
