import os
import sqlite3
from typing import Iterable


def db_path() -> str:
    # Episodic ingest/recall fallback only. Dream proposals and app settings live in APP_DB (Go store).
    return os.getenv("GRAPHITI_SQLITE_PATH", "graphiti_sidecar.db")


def connect() -> sqlite3.Connection:
    conn = sqlite3.connect(db_path(), check_same_thread=False)
    conn.row_factory = sqlite3.Row
    return conn


SCHEMA: Iterable[str] = [
    """
    CREATE TABLE IF NOT EXISTS memory_events (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      conv_id TEXT NOT NULL,
      created_at TEXT NOT NULL,
      role TEXT NOT NULL,
      text TEXT NOT NULL,
      meta_json TEXT NOT NULL DEFAULT '{}'
    );
    """,
    """
    CREATE INDEX IF NOT EXISTS idx_memory_events_conv_created
      ON memory_events (conv_id, created_at);
    """,
]


def init_schema(conn: sqlite3.Connection) -> None:
    cur = conn.cursor()
    for stmt in SCHEMA:
        cur.execute(stmt)
    conn.commit()

