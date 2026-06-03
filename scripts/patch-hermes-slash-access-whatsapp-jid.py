#!/usr/bin/env python3
"""Match WhatsApp multi-device JIDs against allow_admin_from (e.g. :34@s.whatsapp.net)."""
from __future__ import annotations

import re
from pathlib import Path

TARGET = Path("/usr/local/lib/hermes-agent/gateway/slash_access.py")
MARKER = "def _admin_lookup_keys(user_id: Optional[str])"

HELPER = '''
def _admin_lookup_keys(user_id: Optional[str]) -> tuple[str, ...]:
    """Expand WhatsApp/LID JIDs so allowlists can use bare phone or canonical JID."""
    if not user_id:
        return ()
    uid = str(user_id).strip()
    if not uid:
        return ()
    keys: list[str] = [uid]
    m = re.match(r"^(\\d+):\\d+@(s\\.whatsapp\\.net)$", uid)
    if m:
        keys.append(f"{m.group(1)}@{m.group(2)}")
        keys.append(m.group(1))
    m2 = re.match(r"^(\\d+)@(s\\.whatsapp\\.net)$", uid)
    if m2:
        keys.append(m2.group(1))
    m3 = re.match(r"^(\\d+):\\d+@(lid)$", uid)
    if m3:
        keys.append(f"{m3.group(1)}@{m3.group(2)}")
        keys.append(m3.group(1))
    m4 = re.match(r"^(\\d+)@(lid)$", uid)
    if m4:
        keys.append(m4.group(1))
    return tuple(dict.fromkeys(keys))


'''

OLD_IS_ADMIN = """    def is_admin(self, user_id: Optional[str]) -> bool:
        if not self.enabled:
            # Gating disabled → treat every allowed user as admin so
            # downstream code can keep using ``is_admin`` / ``can_run``
            # uniformly.
            return True
        if not user_id:
            return False
        return str(user_id) in self.admin_user_ids"""

NEW_IS_ADMIN = """    def is_admin(self, user_id: Optional[str]) -> bool:
        if not self.enabled:
            # Gating disabled → treat every allowed user as admin so
            # downstream code can keep using ``is_admin`` / ``can_run``
            # uniformly.
            return True
        if not user_id:
            return False
        for key in _admin_lookup_keys(user_id):
            if key in self.admin_user_ids:
                return True
        return False"""

BOTCHED_START = """    user_allowed_commands: FrozenSet[str]

def _admin_lookup_keys(user_id: Optional[str]) -> tuple[str, ...]:"""


def ensure_import_re(text: str) -> str:
    if "\nimport re\n" in text or text.startswith("import re\n"):
        return text
    return text.replace(
        "from typing import Any, FrozenSet, Iterable, Optional, Tuple\n",
        "from typing import Any, FrozenSet, Iterable, Optional, Tuple\n\nimport re\n",
        1,
    )


def fix_botched(text: str) -> str:
    """Repair a prior bad patch that nested methods inside the helper."""
    if BOTCHED_START not in text:
        return text
    # Extract helper body through return tuple(...)
    idx = text.index(BOTCHED_START)
    rest = text[idx + len("    user_allowed_commands: FrozenSet[str]\n\n") :]
    helper_end = rest.index("    def is_admin")
    helper_body = rest[:helper_end]
    methods_and_rest = rest[helper_end:]
    text = (
        text[:idx]
        + "    user_allowed_commands: FrozenSet[str]\n\n"
        + methods_and_rest
    )
    if MARKER not in text:
        text = text.replace(
            "@dataclass(frozen=True)\nclass SlashAccessPolicy:",
            HELPER.lstrip("\n") + "@dataclass(frozen=True)\nclass SlashAccessPolicy:",
            1,
        )
    return text


def patch_is_admin(text: str) -> str:
    text = fix_botched(text)
    text = ensure_import_re(text)
    if MARKER not in text:
        text = text.replace(
            "@dataclass(frozen=True)\nclass SlashAccessPolicy:",
            HELPER.lstrip("\n") + "@dataclass(frozen=True)\nclass SlashAccessPolicy:",
            1,
        )
    if NEW_IS_ADMIN in text:
        return text
    if OLD_IS_ADMIN not in text:
        raise SystemExit("is_admin block not found — hermes-agent may have changed")
    return text.replace(OLD_IS_ADMIN, NEW_IS_ADMIN, 1)


def main() -> None:
    raw = TARGET.read_text(encoding="utf-8")
    new = patch_is_admin(raw)
    if new != raw:
        TARGET.write_text(new, encoding="utf-8")
        print(f"ok: patched {TARGET}")
    else:
        print("ok: already patched")


if __name__ == "__main__":
    main()
