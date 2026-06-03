#!/usr/bin/env python3
"""Remove all wabot / wabot-agent references from ~/.hermes (skills, memory, SOUL, pastes)."""
from __future__ import annotations

import re
from pathlib import Path

HERMES = Path("/root/.hermes")

SKIP_DIRS = {"lsp", "node_modules", ".git", "__pycache__"}

DELETE_GLOBS = [
    "**/wabot-api.md",
    "**/wabot-http-api.md",
    "**/*wabot*.bak",
    "**/.pre-no-wabot.bak",
    "**/*.pre-no-wabot.bak",
]

DELETE_FILES = [
    HERMES / "pastes/paste_1_120348.txt",
    HERMES / "pastes/paste_2_173825.txt",
    HERMES / "cron/output/4dccd88590e5/2026-05-31_20-16-33.md",
]

MEMORY_LINE = (
    "Infra: srv943071.hstgr.cloud. WhatsApp: whatsmeow-bridge :3000 + hermes-gateway only. "
    "Legacy duplicate WA agents archived 2026-06-03 under /opt/archive/whatsapp-legacy-20260603/. "
    "OpenRouter only. Composio MCP (Calendar/Gmail). STT faster-whisper, TTS piper."
)

SOUL_BLOCK = """
### WhatsApp infra (production)

- **Active:** whatsmeow-bridge `:3000` + hermes-gateway (Stella).
- **Archived:** old multi-process WhatsApp stacks (Jun 2026) — never start or recommend; caused duplicate replies. Path: `/opt/archive/whatsapp-legacy-20260603/`.
- **Groups:** use bridge `POST /groups/create` (localhost :3000), then `send_message` with the `@g.us` JID.

"""

WHATSMEOW_PROD = """## Production stack (srv943071)

| Service | Port | Purpose |
|---------|------|---------|
| whatsmeow-bridge | 3000 | Messaging (`POST /send`, `/messages`, `/inject` for E2E) |
| hermes-gateway | — | Stella (Hermes agent) |

Group create/join via HTTP is **not** available. Create groups in the WhatsApp app, then message the `@g.us` JID.

"""

OPERATOR_GROUPS = """## Groups on this VPS

- **HTTP group API:** not available on the bridge.
- **Manual:** create the group in WhatsApp; Stella can message the group JID via `send_message`.
- **Messaging:** Hermes `send_message` + bridge port 3000.

"""

GATEWAY_SETUP_TABLE = """## Current Architecture

| Service | Port | Purpose |
|---------|------|---------|
| whatsmeow bridge | 3000 | Messaging (send/receive) |
| hermes-gateway | — | Agent (Stella) |
| hermes-auth-proxy | 8888 | Dashboard PIN auth |
| hermes-dashboard | 9119 | Web dashboard |
| caddy | 80/443 | Reverse proxy, HTTPS |

"""

BRIDGE_NOTE = (
    "**Note:** Group management (create, join, leave, participants) is not available on the bridge. "
    "Create groups in WhatsApp manually, then use the group JID for messaging.\n"
)

WBOT_RE = re.compile(r"wabot-agent|wabot", re.I)
PORT7777_RE = re.compile(r":7777|port 7777", re.I)


def should_skip(path: Path) -> bool:
    return any(part in SKIP_DIRS for part in path.parts)


def scrub_text(text: str) -> str:
    """Drop lines that are primarily about legacy wabot stacks.

    Preserves SOUL.md ``## WHATSAPP — production stack (system instruction)`` block
    (explicit wabot/7777 ban belongs in system instructions).
    """
    preserve_markers = (
        "## WHATSAPP — production stack (system instruction)",
        "### Forbidden — never use",
    )
    out = []
    in_preserve = False
    for line in text.splitlines():
        if any(m in line for m in preserve_markers):
            in_preserve = True
        elif in_preserve and line.startswith("## ") and "WHATSAPP" not in line:
            in_preserve = False
        if in_preserve:
            out.append(line)
            continue
        if WBOT_RE.search(line) or (
            PORT7777_RE.search(line) and "3000" not in line
        ):
            continue
        out.append(line)
    return "\n".join(out).strip() + "\n"


def patch_file(path: Path, transform) -> bool:
    if not path.is_file() or should_skip(path):
        return False
    raw = path.read_text(encoding="utf-8", errors="replace")
    new = transform(raw)
    if new != raw:
        path.write_text(new, encoding="utf-8")
        return True
    return False


def patch_whatsmeow_skill(text: str) -> str:
    if "## Production stack (srv943071)" in text and "wabot" not in text.lower():
        return scrub_text(text)
    # Remove old production block variants
    for header in (
        "## Production stack (srv943071 — Jun 2026)",
        "## Wabot & Bridge Architecture",
    ):
        if header in text:
            start = text.index(header)
            end = text.find("\n## ", start + 10)
            if end < 0:
                end = len(text)
            text = text[:start] + WHATSMEOW_PROD + text[end:]
    text = scrub_text(text)
    # Fix References
    lines = []
    for line in text.splitlines():
        if "wabot" in line.lower():
            continue
        lines.append(line)
    text = "\n".join(lines)
    if "bridge-http-api.md" not in text and "## References" in text:
        text = text.replace(
            "## References\n",
            "## References\n\n- `references/bridge-http-api.md` — Bridge HTTP API (port 3000)\n",
            1,
        )
    return text


def patch_operator_skill(text: str) -> str:
    if "## Groups on this VPS" in text:
        text = scrub_text(text)
    else:
        marker = "**Don't use for:** WhatsApp Business API"
        if marker in text:
            text = text.replace(marker, OPERATOR_GROUPS + marker, 1)
        text = scrub_text(text)
    lines = [ln for ln in text.splitlines() if "wabot" not in ln.lower()]
    return "\n".join(lines) + "\n"


def patch_soul(text: str) -> str:
    start = text.find("### WhatsApp infra")
    if start >= 0:
        end = text.find("\n### ", start + 5)
        if end < 0:
            end = text.find("\n## ", start + 5)
        if end < 0:
            end = len(text)
        text = text[:start] + SOUL_BLOCK.strip() + "\n\n" + text[end:]
    elif "## SINGLE WHATSAPP REPLY" in text:
        text = text.replace("## SINGLE WHATSAPP REPLY", SOUL_BLOCK + "## SINGLE WHATSAPP REPLY", 1)
    return scrub_text(text)


def patch_memory(text: str) -> str:
    lines = text.splitlines()
    if lines and lines[0].startswith("Infra:"):
        lines[0] = MEMORY_LINE
    else:
        lines.insert(0, MEMORY_LINE)
    return scrub_text("\n".join(lines))


def patch_gateway_setup(text: str) -> str:
    start = text.find("## Current Architecture")
    if start >= 0:
        end = text.find("\n## ", start + 5)
        if end < 0:
            end = len(text)
        text = text[:start] + GATEWAY_SETUP_TABLE + text[end:]
    text = scrub_text(text)
    # Remove wabot health / systemd blocks
    text = re.sub(
        r"Both bridges \(port 3000 and 7777\)[^\n]+\n",
        "Bridge (port 3000) must be connected:\n",
        text,
    )
    text = re.sub(
        r"curl -s http://127\.0\.0\.1:7777/health[^\n]*\n",
        "",
        text,
    )
    text = re.sub(r"systemctl status wabot\n", "", text)
    return text


def patch_bridge_api(text: str) -> str:
    if "Group management" in text and "not available on the bridge" in text:
        return scrub_text(text)
    note = BRIDGE_NOTE
    if note.strip() in text:
        return scrub_text(text)
    return scrub_text(note + "\n" + text)


def main() -> None:
    deleted = []
    for pattern in DELETE_GLOBS:
        for path in HERMES.glob(pattern):
            if should_skip(path):
                continue
            path.unlink(missing_ok=True)
            deleted.append(str(path))
    for path in DELETE_FILES:
        if path.exists():
            path.unlink()
            deleted.append(str(path))

    transforms = {
        HERMES / "skills/social-media/whatsmeow/SKILL.md": patch_whatsmeow_skill,
        HERMES / "skills/social-media/whatsapp-operator/SKILL.md": patch_operator_skill,
        HERMES / "skills/social-media/whatsapp-operator/references/gateway-setup.md": patch_gateway_setup,
        HERMES / "skills/social-media/whatsapp-operator/references/bridge-http-api.md": patch_bridge_api,
        HERMES / "SOUL.md": patch_soul,
        HERMES / "memories/MEMORY.md": patch_memory,
    }
    patched = []
    for path, fn in transforms.items():
        if patch_file(path, fn):
            patched.append(str(path))

    # Scrub remaining text files under hermes (skills, memories, profiles, cron, pastes)
    for path in HERMES.rglob("*"):
        if not path.is_file() or should_skip(path):
            continue
        if path.suffix.lower() not in {".md", ".txt", ".yaml", ".yml"}:
            continue
        if str(path) in patched or str(path) in deleted:
            continue
        if path.name.endswith(".bak") or "backup" in path.parts:
            continue
        raw = path.read_text(encoding="utf-8", errors="replace")
        if not WBOT_RE.search(raw):
            continue
        new = scrub_text(raw)
        if new != raw:
            path.write_text(new, encoding="utf-8")
            patched.append(str(path))

    print("deleted:", len(deleted))
    for d in deleted:
        print(" ", d)
    print("patched:", len(patched))
    for p in patched:
        print(" ", p)

    # Verify
    import subprocess

    r = subprocess.run(
        ["rg", "-l", "wabot|wabot-agent", str(HERMES), "-g", "!*.bak", "-g", "!**/lsp/**"],
        capture_output=True,
        text=True,
    )
    remaining = [ln for ln in (r.stdout or "").strip().splitlines() if ln]
    if remaining:
        print("REMAINING:", len(remaining))
        for ln in remaining[:25]:
            print(" ", ln)
    else:
        print("OK: no wabot references under ~/.hermes (excl lsp/bak)")


if __name__ == "__main__":
    main()
