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
- `POST /dreams/propose`
  - creates a draft “dream proposal” row in local SQLite (for Go/UI to persist/consume later)

## Notes

- If `NEO4J_URI` is **not** set, `/ingest` and `/recall` operate in a simple SQLite mode (string search over ingested events).
- If `NEO4J_URI` **is** set, the service will attempt to initialize Graphiti on startup; if that fails, it falls back to SQLite mode and reports `graphiti_enabled=false` at `GET /health`.

