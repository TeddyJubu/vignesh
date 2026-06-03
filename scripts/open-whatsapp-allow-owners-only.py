#!/usr/bin/env python3
"""Open WhatsApp DMs/groups to everyone; keep admin/owner on Teddy + Vignesh only."""
import re
from pathlib import Path

HERMES = Path("/root/.hermes")
CONFIG = HERMES / "config.yaml"
ENV = HERMES / ".env"

OWNERS = [
    "6590013157",
    "6590013157@s.whatsapp.net",
    "6590013157:31@s.whatsapp.net",
    "56676572987400@lid",
    "56676572987400:31@lid",
    "8801521207499",
    "8801521207499@s.whatsapp.net",
    "8801521207499:34@s.whatsapp.net",
    "87192181436622@lid",
    "87192181436622:34@lid",
]


def yaml_entries(key: str, items: list[str]) -> str:
    lines = [f"  {key}:"]
    for item in items:
        lines.append(f'    - "{item}"')
    return "\n".join(lines)


def main() -> None:
    block = (
        "whatsapp:\n"
        "  gateway_restart_notification: false\n"
        "  group_sessions_per_user: true\n"
        "  dm_policy: open\n"
        "  group_policy: open\n"
        f"{yaml_entries('allow_from', OWNERS)}\n"
        f"{yaml_entries('allow_admin_from', OWNERS)}\n"
        f"{yaml_entries('group_allow_from', OWNERS)}\n"
        f"{yaml_entries('group_allow_admin_from', OWNERS)}\n"
    )
    text = CONFIG.read_text()
    text = re.sub(r"^whatsapp:\n(?:  .+\n)*", block, text, count=1, flags=re.M)
    CONFIG.write_text(text)

    env = ENV.read_text()
    env = re.sub(r"^WHATSAPP_ALLOWED_USERS=.*$", "WHATSAPP_ALLOWED_USERS=*", env, flags=re.M)
    if re.search(r"^WHATSAPP_DM_POLICY=", env, flags=re.M):
        env = re.sub(r"^WHATSAPP_DM_POLICY=.*$", "WHATSAPP_DM_POLICY=open", env, flags=re.M)
    else:
        env += "\nWHATSAPP_DM_POLICY=open\n"
    ENV.write_text(env)
    print("ok: open DMs/groups; admin only for Teddy + Vignesh")


if __name__ == "__main__":
    main()
