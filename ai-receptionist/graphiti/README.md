# Graphiti sidecar (memory service)

Minimal FastAPI sidecar for **ingest** + **recall** backed by Graphiti (getzep/graphiti) when configured, with a lightweight SQLite fallback so the service can start even without Neo4j.

## Requirements

- Python 3.10+
- (Optional, for Graphiti mode) Neo4j reachable at `NEO4J_URI`
- (Optional, for Graphiti mode) OpenAI key for Graphiti inference: `OPENAI_API_KEY`

## Run (dev)

```bash
cd ai-receptionist/graphiti
python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt

# Optional Graphiti mode:
# export NEO4J_URI="bolt://localhost:7687"
# export NEO4J_USER="neo4j"
# export NEO4J_PASSWORD="password"
# export OPENAI_API_KEY="..."

uvicorn app.main:app --reload --host 0.0.0.0 --port 8333
```

## API

- `POST /ingest`
  - body: `{ "conv_id": "...", "timestamp": "...", "role": "user|assistant", "text": "...", "meta": {...} }`
- `GET /recall?conv_id=...&q=...`
  - returns: `{ "items": [{ "text": "...", "score": 0.0, "source": "graphiti|sqlite", "created_at": "..." }], "snippet": "..." }`
  - Graphiti ingest/recall use `group_id=conv_id` so memories stay scoped per conversation.
- `POST /dreams/propose`
  - returns a draft proposal payload (`id`, `title`, `patch`, `rationale`). **Does not persist** — the Go app writes proposals into `APP_DB` (`database.db`) via `POST /api/dreams/propose`.

## Environment

| Variable | Purpose |
|----------|---------|
| `GRAPHITI_SQLITE_PATH` | Sidecar episodic fallback DB (default `graphiti_sidecar.db`). Not `APP_DB`. |
| `NEO4J_URI`, `NEO4J_USER`, `NEO4J_PASSWORD` | Optional Graphiti graph mode |
| `OPENAI_API_KEY` | Optional, for Graphiti inference |

Go process (same repo):

| Variable | Purpose |
|----------|---------|
| `APP_DB` | Main SQLite (`database.db`): settings, dreams, conversations |
| `GRAPHITI_URL` | Sidecar base URL (e.g. `http://127.0.0.1:8333`) for ingest/recall/propose |

## Notes

- Dream proposals and `app_settings` live only in **`APP_DB`** (`internal/store`), not in the sidecar SQLite.
- If `NEO4J_URI` is **not** set, `/ingest` and `/recall` operate in a simple SQLite mode (string search over ingested events in `GRAPHITI_SQLITE_PATH`).
- If `NEO4J_URI` **is** set, the service will attempt to initialize Graphiti on startup; if that fails, it falls back to SQLite mode and reports `graphiti_enabled=false` at `GET /health`.

