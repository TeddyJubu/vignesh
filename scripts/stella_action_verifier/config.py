"""Load ~/.hermes/action-verifier.yaml with defaults."""
from __future__ import annotations

import json
import os
from pathlib import Path

DEFAULTS = {
    "enabled": True,
    "platforms": ["whatsapp"],
    "model": "google/gemma-3-4b-it",
    "use_llm": "on_fail",  # always | on_fail | never
    "max_response_chars": 4000,
    "evidence_window_sec": 300,
    "inject_correction": True,
    "log_path": "~/.hermes/logs/action-verifier.log",
}


def load_config() -> dict:
    home = Path(os.environ.get("HERMES_HOME", Path.home() / ".hermes"))
    cfg = dict(DEFAULTS)
    for name in ("action-verifier.json", "action-verifier.yaml"):
        path = home / name
        if not path.is_file():
            continue
        raw = path.read_text(encoding="utf-8")
        if name.endswith(".json"):
            loaded = json.loads(raw)
        else:
            try:
                import yaml  # type: ignore

                loaded = yaml.safe_load(raw) or {}
            except Exception:
                loaded = {}
        if isinstance(loaded, dict):
            cfg.update(loaded)
        break
    cfg["log_path"] = str(Path(os.path.expanduser(str(cfg["log_path"]))))
    return cfg
