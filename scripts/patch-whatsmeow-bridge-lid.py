#!/usr/bin/env python3
"""Add LID→phone fallback to whatsmeow-bridge main.go."""
from pathlib import Path

MAIN = Path("/opt/whatsmeow-bridge/main.go")


def main() -> None:
    text = MAIN.read_text()
    if "lidFallbackPN" in text:
        print("already patched")
        return

    text = text.replace(
        "var (\n\tclient       *whatsmeow.Client",
        """var (
\t// When WhatsApp only exposes @lid JIDs, map known LIDs to E.164 user parts.
\tlidFallbackPN = map[string]string{
\t\t"87192181436622": "8801521207499", // Teddy
\t\t"56676572987400": "6590013157",   // Vignesh
\t}
\tclient       *whatsmeow.Client""",
        1,
    )

    helper = """

func resolvePNJID(jid types.JID) types.JID {
\tif jid.Server != types.HiddenUserServer {
\t\treturn jid
\t}
\tif client != nil && client.Store != nil && client.Store.LIDs != nil {
\t\tif resolved, err := client.Store.LIDs.GetPNForLID(context.Background(), jid); err == nil && !resolved.IsEmpty() {
\t\t\treturn resolved
\t\t}
\t}
\tif phone, ok := lidFallbackPN[jid.User]; ok && phone != "" {
\t\treturn types.NewJID(phone, types.DefaultUserServer)
\t}
\treturn jid
}
"""
    text = text.replace("func handleEvent(evt any) {", helper + "\nfunc handleEvent(evt any) {", 1)

    old = """\t\t// Resolve LID to phone number if needed
\t\tsenderPN := senderJID
\t\tif senderJID.Server == types.HiddenUserServer {
\t\t\tif resolved, err := client.Store.LIDs.GetPNForLID(context.Background(), senderJID); err == nil && !resolved.IsEmpty() {
\t\t\t\tsenderPN = resolved
\t\t\t}
\t\t}

\t\t// Also resolve chat JID if it's a LID
\t\tchatPN := chatJID
\t\tif chatJID.Server == types.HiddenUserServer {
\t\t\tif resolved, err := client.Store.LIDs.GetPNForLID(context.Background(), chatJID); err == nil && !resolved.IsEmpty() {
\t\t\t\tchatPN = resolved
\t\t\t}
\t\t}"""
    new = "\t\tsenderPN := resolvePNJID(senderJID)\n\t\tchatPN := resolvePNJID(chatJID)"
    if old not in text:
        raise SystemExit("lid resolve block missing")
    text = text.replace(old, new, 1)
    MAIN.write_text(text)
    print("patched", MAIN)


if __name__ == "__main__":
    main()
