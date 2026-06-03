#!/usr/bin/env python3
"""Fix stale 'connected' health + auto-reconnect on WhatsApp socket drop."""
from __future__ import annotations

from pathlib import Path

MAIN = Path("/opt/whatsmeow-bridge/main.go")
MARKER = "stella: connection health"


def main() -> None:
    text = MAIN.read_text(encoding="utf-8")
    if MARKER in text:
        print("ok: bridge connection-health already patched")
        return

    if "func handleEvent(evt any)" not in text:
        raise SystemExit("handleEvent not found")

    # --- reconnect helper (before handleEvent) ---
    if "func tryReconnect()" not in text:
        text = text.replace(
            "func handleEvent(evt any) {",
            '''func tryReconnect() {
\tif client == nil || client.Store == nil || client.Store.ID == nil {
\t\treturn
\t}
\tif client.IsConnected() {
\t\tconnState = "connected"
\t\treturn
\t}
\tconnState = "connecting"
\tlog.Printf("🔄 Reconnecting WhatsApp socket...")
\tif err := client.Connect(); err != nil {
\t\tlog.Printf("❌ Reconnect failed: %v", err)
\t\tconnState = "disconnected"
\t\treturn
\t}
\tconnState = "connected"
\tlog.Printf("✅ WhatsApp reconnected as %s", client.Store.ID.User)
}

func markSendDisconnected(err error) {
\tif err == nil {
\t\treturn
\t}
\tmsg := strings.ToLower(err.Error())
\tif strings.Contains(msg, "websocket not connected") || strings.Contains(msg, "not logged in") {
\t\tconnState = "disconnected"
\t\tgo tryReconnect()
\t}
}

func handleEvent(evt any) {''',
            1,
        )

    # --- Connected / Disconnected events ---
    if "case *events.Connected:" not in text:
        text = text.replace(
            "\tswitch v := evt.(type) {\n\tcase *events.Message:",
            f"""\tswitch v := evt.(type) {{
\tcase *events.Connected: // {MARKER}
\t\tconnState = "connected"
\t\tlog.Printf("✅ WhatsApp socket connected")
\tcase *events.Disconnected: // {MARKER}
\t\tconnState = "disconnected"
\t\tlog.Printf("⚠️ WhatsApp socket disconnected — scheduling reconnect")
\t\tgo tryReconnect()
\tcase *events.Message:""",
            1,
        )

    # --- health exposes real socket state ---
    old_health = '''\tw.Header().Set("Content-Type", "application/json")
\tjsonResponse(w, map[string]interface{}{
\t\t"status":      connState,
\t\t"queueLength": qLen,
\t\t"uptime":      time.Since(startTime).Seconds(),
\t\t"backend":     "whatsmeow",
\t})'''

    new_health = '''\tw.Header().Set("Content-Type", "application/json")
\tsocketOK := client != nil && client.IsConnected()
\tsendReady := connState == "connected" && socketOK
\tjsonResponse(w, map[string]interface{}{
\t\t"status":          connState,
\t\t"sendReady":       sendReady,
\t\t"socketConnected": socketOK,
\t\t"queueLength":     qLen,
\t\t"uptime":          time.Since(startTime).Seconds(),
\t\t"backend":         "whatsmeow",
\t}) // stella: connection health'''

    if old_health not in text:
        raise SystemExit("handleHealth body not found")
    text = text.replace(old_health, new_health, 1)

    # --- gate sends on real socket ---
    text = text.replace(
        '\tif connState != "connected" {\n\t\tjsonError(w, "WhatsApp not connected", 503)\n\t\treturn\n\t}',
        '\tif connState != "connected" || client == nil || !client.IsConnected() {\n\t\tjsonError(w, "WhatsApp not connected", 503)\n\t\treturn\n\t}',
    )

    # --- send failures mark disconnected ---
    text = text.replace(
        '\tif err != nil {\n\t\tjsonError(w, "Send failed: "+err.Error(), 500)',
        '\tif err != nil {\n\t\tmarkSendDisconnected(err)\n\t\tjsonError(w, "Send failed: "+err.Error(), 500)',
        1,
    )
    text = text.replace(
        '\tif err != nil {\n\t\tjsonError(w, "Upload failed: "+err.Error(), 500)',
        '\tif err != nil {\n\t\tmarkSendDisconnected(err)\n\t\tjsonError(w, "Upload failed: "+err.Error(), 500)',
        1,
    )

    bak = MAIN.with_suffix(".go.pre-connection-health.bak")
    if not bak.exists():
        bak.write_text(MAIN.read_text(encoding="utf-8"), encoding="utf-8")
    MAIN.write_text(text, encoding="utf-8")
    print("ok: bridge connection-health patched — run: cd /opt/whatsmeow-bridge && go build -o whatsmeow-bridge .")


if __name__ == "__main__":
    main()
