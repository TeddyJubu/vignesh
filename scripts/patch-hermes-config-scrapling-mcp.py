#!/usr/bin/env python3
"""Register Scrapling MCP server in ~/.hermes/config.yaml."""
from __future__ import annotations

from pathlib import Path

CONFIG = Path("/root/.hermes/config.yaml")
SCRAPLING_BIN = "/usr/local/lib/hermes-agent/venv/bin/scrapling"
MARKER = "  scrapling:"

BLOCK = f"""  scrapling:
    command: "{SCRAPLING_BIN}"
    args:
      - mcp
    timeout: 300
    connect_timeout: 90
    supports_parallel_tool_calls: true
"""


def main() -> None:
    text = CONFIG.read_text(encoding="utf-8")
    if MARKER in text:
        print("ok: scrapling MCP already in config.yaml")
        return
    needle = "mcp_servers:\n"
    if needle not in text:
        raise SystemExit("mcp_servers: not found in config.yaml")
    # Insert after mcp_servers: line, before first existing server
    text = text.replace(needle, needle + BLOCK, 1)
    CONFIG.write_text(text, encoding="utf-8")
    print("ok: added scrapling MCP to config.yaml — restart hermes-gateway")


if __name__ == "__main__":
    main()
