from __future__ import annotations

import json
import uuid
from contextlib import asynccontextmanager
from typing import Optional

from fastapi import BackgroundTasks, FastAPI, HTTPException, Query, Request

from .db import connect, init_schema
from .graphiti_client import GraphitiState, init_graphiti, now_rfc3339
from .models import (
    DreamProposeRequest,
    DreamProposeResponse,
    IngestRequest,
    RecallItem,
    RecallResponse,
)


@asynccontextmanager
async def lifespan(app: FastAPI):
    conn = connect()
    init_schema(conn)
    app.state.db = conn
    app.state.graphiti = await init_graphiti()
    yield
    try:
        g: GraphitiState = app.state.graphiti
        if g.enabled and g.client is not None:
            close = getattr(g.client, "close", None)
            if callable(close):
                maybe = close()
                # Graphiti close may be async or sync; tolerate either.
                if hasattr(maybe, "__await__"):
                    await maybe
    finally:
        conn.close()


app = FastAPI(title="ai-receptionist Graphiti sidecar", version="0.1.0", lifespan=lifespan)


@app.get("/health")
async def health(request: Request):
    g: GraphitiState = request.app.state.graphiti
    return {
        "ok": True,
        "graphiti_enabled": g.enabled,
        "graphiti_error": g.error,
    }


@app.post("/ingest")
async def ingest(request: Request, body: IngestRequest):
    conn = request.app.state.db
    meta_json = json.dumps(body.meta or {}, ensure_ascii=False)
    conn.execute(
        "INSERT INTO memory_events (conv_id, created_at, role, text, meta_json) VALUES (?, ?, ?, ?, ?)",
        (body.conv_id, body.timestamp, body.role, body.text, meta_json),
    )
    conn.commit()

    g: GraphitiState = request.app.state.graphiti
    if g.enabled and g.client is not None:
        # Best-effort: store an episode with minimal metadata.
        try:
            from graphiti_core.nodes import EpisodeType  # type: ignore

            await g.client.add_episode(  # type: ignore[attr-defined]
                name=f"{body.conv_id}:{body.role}",
                episode_body=body.text,
                source=EpisodeType.text,
                source_description="ai-receptionist turn event",
                reference_time=None,
            )
        except Exception:
            # Keep ingest robust; SQLite is authoritative for sidecar even if graphiti fails.
            pass

    return {"ok": True}


def _sqlite_recall(conn, conv_id: str, q: str, limit: int) -> list[RecallItem]:
    q_like = f"%{q.strip()}%"
    rows = conn.execute(
        """
        SELECT created_at, text
        FROM memory_events
        WHERE conv_id = ?
          AND (? = '' OR text LIKE ?)
        ORDER BY created_at DESC
        LIMIT ?
        """,
        (conv_id, q.strip(), q_like, limit),
    ).fetchall()
    return [
        RecallItem(text=r["text"], score=0.0, source="sqlite", created_at=r["created_at"])
        for r in rows
    ]


@app.get("/recall", response_model=RecallResponse)
async def recall(
    request: Request,
    conv_id: str = Query(..., min_length=1),
    q: str = Query("", max_length=500),
    limit: int = Query(5, ge=1, le=20),
):
    conn = request.app.state.db
    g: GraphitiState = request.app.state.graphiti

    items: list[RecallItem] = []
    if g.enabled and g.client is not None and q.strip():
        try:
            # Graphiti search APIs can evolve; keep this best-effort and fall back.
            results = await g.client.search(q.strip())  # type: ignore[attr-defined]
            for r in (results or [])[:limit]:
                text = getattr(r, "fact", None) or getattr(r, "text", None) or str(r)
                score = float(getattr(r, "score", 0.0) or 0.0)
                items.append(RecallItem(text=text, score=score, source="graphiti"))
        except Exception:
            items = []

    if not items:
        items = _sqlite_recall(conn, conv_id, q, limit)

    snippet = "\n".join(f"- {it.text.strip()}" for it in items if it.text.strip())
    snippet = snippet[:2000]  # hard bound for prompt injection safety
    return RecallResponse(items=items, snippet=snippet)


def _insert_dream(conn, title: str, patch: dict, rationale: str) -> str:
    pid = str(uuid.uuid4())
    conn.execute(
        "INSERT INTO dream_proposals (id, created_at, status, title, patch, rationale) VALUES (?, ?, ?, ?, ?, ?)",
        (pid, now_rfc3339(), "proposed", title, json.dumps(patch, ensure_ascii=False), rationale),
    )
    conn.commit()
    return pid


def _dream_background(conn, title: str, patch: dict, rationale: str):
    _insert_dream(conn, title, patch, rationale)


@app.post("/dreams/propose", response_model=DreamProposeResponse)
async def dreams_propose(request: Request, body: DreamProposeRequest, bg: BackgroundTasks):
    # “Background job or endpoint”: keep it endpoint-driven; do work in background.
    conn = request.app.state.db
    if not body.patch:
        # Minimal default patch shape so UI/go can store something meaningful.
        body.patch = {"target": "identity_soul", "content": "DRAFT: (fill in) Proposed update."}
    pid = str(uuid.uuid4())
    bg.add_task(
        _dream_background,
        conn,
        body.title,
        body.patch,
        body.rationale,
    )
    return DreamProposeResponse(id=pid, status="queued")

