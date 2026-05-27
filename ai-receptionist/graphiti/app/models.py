from __future__ import annotations

from typing import Any, Literal, Optional

from pydantic import BaseModel, Field


class IngestRequest(BaseModel):
    conv_id: str = Field(..., min_length=1)
    timestamp: str = Field(..., min_length=1)  # RFC3339-ish; stored as-is
    role: Literal["user", "assistant"]
    text: str = Field(..., min_length=1)
    meta: Optional[dict[str, Any]] = None


class RecallItem(BaseModel):
    text: str
    score: float = 0.0
    source: Literal["sqlite", "graphiti"] = "sqlite"
    created_at: Optional[str] = None


class RecallResponse(BaseModel):
    items: list[RecallItem]
    snippet: str


class DreamProposeRequest(BaseModel):
    conv_id: str = Field(..., min_length=1)
    title: str = Field(default="Dream proposal", min_length=1)
    rationale: str = Field(default="Automated draft proposal (not applied).", min_length=1)
    patch: dict[str, Any] = Field(default_factory=dict)


class DreamProposeResponse(BaseModel):
    id: str
    status: str

