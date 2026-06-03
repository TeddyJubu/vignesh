#!/usr/bin/env bash
# Grant Teddy co-owner / full admin on Hermes WhatsApp (vignesh VPS).
set -euo pipefail

HERMES_DIR="${HERMES_DIR:-/root/.hermes}"
ENV_FILE="$HERMES_DIR/.env"
CONFIG="$HERMES_DIR/config.yaml"
SOUL="$HERMES_DIR/SOUL.md"
MEMORY="$HERMES_DIR/memories/MEMORY.md"
USER_MEM="$HERMES_DIR/memories/USER.md"

OWNERS=(
  "6590013157"
  "6590013157@s.whatsapp.net"
  "6590013157:31@s.whatsapp.net"
  "56676572987400@lid"
  "56676572987400:31@lid"
  "8801521207499"
  "8801521207499@s.whatsapp.net"
  "87192181436622@lid"
  "87192181436622:34@lid"
)

python3 << "PY"
import re
from pathlib import Path

HERMES_DIR = Path("/root/.hermes")
ENV_FILE = HERMES_DIR / ".env"
CONFIG = HERMES_DIR / "config.yaml"
SOUL = HERMES_DIR / "SOUL.md"
MEMORY = HERMES_DIR / "memories/MEMORY.md"
USER_MEM = HERMES_DIR / "memories/USER.md"

owners = [
    "6590013157",
    "6590013157@s.whatsapp.net",
    "6590013157:31@s.whatsapp.net",
    "56676572987400@lid",
    "56676572987400:31@lid",
    "8801521207499",
    "8801521207499@s.whatsapp.net",
    "87192181436622@lid",
    "87192181436622:34@lid",
]
owners_csv = "*"

# --- .env ---
env = ENV_FILE.read_text()
if "WHATSAPP_ALLOWED_USERS=*" in env or 'WHATSAPP_ALLOWED_USERS="*"' in env:
    env = re.sub(
        r"^WHATSAPP_ALLOWED_USERS=.*$",
        f"WHATSAPP_ALLOWED_USERS={owners_csv}",
        env,
        flags=re.M,
    )
else:
    env = re.sub(
        r"^WHATSAPP_ALLOWED_USERS=.*$",
        f"WHATSAPP_ALLOWED_USERS={owners_csv}",
        env,
        flags=re.M,
    )
if "WHATSAPP_HOME_CHANNEL_NAME=" not in env:
    env += "\nWHATSAPP_HOME_CHANNEL_NAME=Teddy (dev home)\n"
ENV_FILE.write_text(env)
print("updated .env WHATSAPP_ALLOWED_USERS")

# --- config.yaml whatsapp block ---
cfg = CONFIG.read_text()
wa_block = """whatsapp:
  gateway_restart_notification: false
  group_sessions_per_user: true
  dm_policy: open
  group_policy: open
  allow_from:
    - \"6590013157\"
    - \"6590013157@s.whatsapp.net\"
    - \"6590013157:31@s.whatsapp.net\"
    - \"56676572987400@lid\"
    - \"56676572987400:31@lid\"
    - \"8801521207499\"
    - \"8801521207499@s.whatsapp.net\"
    - \"87192181436622@lid\"
    - \"87192181436622:34@lid\"
  allow_admin_from:
    - \"6590013157\"
    - \"6590013157@s.whatsapp.net\"
    - \"6590013157:31@s.whatsapp.net\"
    - \"56676572987400@lid\"
    - \"56676572987400:31@lid\"
    - \"8801521207499\"
    - \"8801521207499@s.whatsapp.net\"
    - \"87192181436622@lid\"
    - \"87192181436622:34@lid\"
  group_allow_from:
    - \"6590013157\"
    - \"6590013157@s.whatsapp.net\"
    - \"8801521207499\"
    - \"8801521207499@s.whatsapp.net\"
    - \"56676572987400@lid\"
    - \"56676572987400:31@lid\"
    - \"87192181436622@lid\"
    - \"87192181436622:34@lid\"
  group_allow_admin_from:
    - \"6590013157\"
    - \"6590013157@s.whatsapp.net\"
    - \"8801521207499\"
    - \"8801521207499@s.whatsapp.net\"
    - \"56676572987400@lid\"
    - \"56676572987400:31@lid\"
    - \"87192181436622@lid\"
    - \"87192181436622:34@lid\"
"""
if re.search(r"^whatsapp:\n", cfg, re.M):
    cfg = re.sub(
        r"^whatsapp:\n(?:  .+\n)*",
        wa_block,
        cfg,
        count=1,
        flags=re.M,
    )
else:
    cfg = cfg.replace("telegram:\n", wa_block + "telegram:\n", 1)
CONFIG.write_text(cfg)
print("updated config.yaml whatsapp admins")

# --- SOUL ---
soul = SOUL.read_text()
soul = soul.replace(
    "### Admin/Developer Mode (Vignesh + Teddy)",
    "### Admin/Developer Mode — Co-owners (Vignesh + Teddy)",
)
soul = soul.replace(
    "- Teddy's WhatsApp: +880 1521 207499 (developer)",
    "- Teddy's WhatsApp: +880 1521 207499 (8801521207499@s.whatsapp.net) — **co-owner**, same admin powers as Vignesh: all slash commands, full transparency, config/tests, third-party send_message.",
)
if "Co-owners" not in soul.split("When talking to ANYONE")[0]:
    soul = soul.replace(
        "When talking to ANYONE who is not Vignesh or Teddy:",
        "Co-owners: **Vignesh** and **Teddy** have identical admin/owner rights on this Hermes instance.\n\nWhen talking to ANYONE who is not Vignesh or Teddy:",
    )
SOUL.write_text(soul)
print("updated SOUL.md")

# --- MEMORY (compact) ---
mem = MEMORY.read_text()
line = "Owners/admins: Vignesh 6590013157 + Teddy 8801521207499 — full slash cmds, admin mode, tests."
if "Teddy 8801521207499" not in mem:
    mem = mem.replace("Stella:", f"{line} Stella:", 1)
    if len(mem) > 2200:
        mem = mem.replace("Guide: skills/devops/hermes-model-routing/references/cost-optimization-guide.md", "Guide: skills/.../cost-optimization-guide.md")
    MEMORY.write_text(mem[:2200] if len(mem) > 2200 else mem)
    print("updated MEMORY.md", len(MEMORY.read_text()))

user = USER_MEM.read_text()
if "co-owner" not in user.lower():
    user = "Teddy: co-owner with Vignesh — full Hermes/WhatsApp admin (tests, config, slash commands, third-party DMs). Same transparency as Vignesh.\n§\n" + user
    USER_MEM.write_text(user[:1375] if len(user) > 1375 else user)
    print("updated USER.md")

PY

systemctl restart hermes-gateway
sleep 2
systemctl is-active hermes-gateway
echo "Done. Teddy + Vignesh are allow_admin_from on WhatsApp."
