from __future__ import annotations

import os
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Optional


@dataclass
class GraphitiState:
    enabled: bool
    error: Optional[str] = None
    client: Optional[object] = None


async def init_graphiti() -> GraphitiState:
    """
    Graphiti requires Neo4j + an LLM key. For local dev/smoke, we allow running
    without it (SQLite fallback).
    """
    neo4j_uri = os.getenv("NEO4J_URI", "").strip()
    if not neo4j_uri:
        return GraphitiState(enabled=False, error="NEO4J_URI not set")

    try:
        from graphiti_core import Graphiti  # type: ignore
    except Exception as e:  # pragma: no cover
        return GraphitiState(enabled=False, error=f"graphiti import failed: {e!r}")

    user = os.getenv("NEO4J_USER", "neo4j").strip()
    password = os.getenv("NEO4J_PASSWORD", "").strip()
    if not password:
        return GraphitiState(enabled=False, error="NEO4J_PASSWORD not set")

    try:
        g = Graphiti(neo4j_uri, user, password)
        # Build indices (idempotent) - only once, but safe to attempt.
        g.build_indices_and_constraints()
        return GraphitiState(enabled=True, client=g)
    except Exception as e:
        return GraphitiState(enabled=False, error=f"graphiti init failed: {e!r}")


def now_rfc3339() -> str:
    return datetime.now(timezone.utc).isoformat()

