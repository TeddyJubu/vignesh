"""Detect actionable claims in assistant replies."""
from __future__ import annotations

import re

# Outbound WhatsApp / messaging
SEND_CLAIM_RE = re.compile(
    r"\b("
    r"sent|delivered|messaged|texted|pinged|asked\s+(him|her|them)|"
    r"reached\s+out|message\s+went\s+through|it's\s+actually\s+delivered"
    r")\b",
    re.I,
)

FILE_CLAIM_RE = re.compile(
    r"\b(sent\s+the\s+(csv|file)|attached|check\s+your\s+whatsapp\s+for)\b",
    re.I,
)

BOOKING_CLAIM_RE = re.compile(
    r"\b(booked|calendar\s+invite\s+sent|event\s+created|locked\s+in\s+for)\b",
    re.I,
)

PHONE_RE = re.compile(
    r"(?:\+?\d[\d\s\-]{7,}\d|"
    r"\d{8,15}@s\.whatsapp\.net)",
    re.I,
)


def extract_phones(text: str) -> list[str]:
    out: list[str] = []
    for m in PHONE_RE.finditer(text or ""):
        raw = re.sub(r"[\s\-]", "", m.group(0))
        if "@" in raw:
            out.append(raw.lower())
        else:
            digits = raw.lstrip("+")
            if len(digits) >= 8:
                out.append(f"{digits}@s.whatsapp.net")
    return list(dict.fromkeys(out))


def has_send_claim(text: str) -> bool:
    return bool(SEND_CLAIM_RE.search(text or ""))


def has_file_claim(text: str) -> bool:
    return bool(FILE_CLAIM_RE.search(text or ""))


def has_booking_claim(text: str) -> bool:
    return bool(BOOKING_CLAIM_RE.search(text or ""))


def needs_verification(response: str, user_message: str) -> bool:
    blob = f"{response}\n{user_message}"
    if has_send_claim(blob) or has_file_claim(blob) or has_booking_claim(blob):
        return True
    # Owner asked to message someone else
    if re.search(r"\b(message|text|ask|contact)\b", user_message or "", re.I):
        if extract_phones(user_message):
            return True
    return False
