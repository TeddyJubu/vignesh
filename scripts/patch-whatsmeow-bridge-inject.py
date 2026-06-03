#!/usr/bin/env python3
"""Add localhost-only POST /inject for E2E synthetic inbound messages."""
from pathlib import Path

MAIN = Path("/opt/whatsmeow-bridge/main.go")


def main() -> None:
    text = MAIN.read_text()
    if "handleInject" in text:
        print("already patched")
        return

    if "mux.HandleFunc(\"/inject\"" not in text:
        text = text.replace(
            'mux.HandleFunc("/messages", handleMessages)',
            'mux.HandleFunc("/messages", handleMessages)\n\tmux.HandleFunc("/inject", handleInject)',
            1,
        )

    handler = '''
func handleInject(w http.ResponseWriter, r *http.Request) {
\tif r.Method != "POST" {
\t\thttp.Error(w, "Method not allowed", 405)
\t\treturn
\t}
\thost, _, err := net.SplitHostPort(r.RemoteAddr)
\tif err != nil {
\t\thost = r.RemoteAddr
\t}
\tif host != "127.0.0.1" && host != "::1" && host != "localhost" {
\t\tjsonError(w, "inject only allowed from localhost", 403)
\t\treturn
\t}

\tvar req map[string]interface{}
\tif err := json.NewDecoder(r.Body).Decode(&req); err != nil {
\t\tjsonError(w, "Invalid JSON", 400)
\t\treturn
\t}

\tbody, _ := req["body"].(string)
\tif strings.TrimSpace(body) == "" {
\t\tjsonError(w, "body required", 400)
\t\treturn
\t}

\tchatID, _ := req["chatId"].(string)
\tsenderID, _ := req["senderId"].(string)
\tif chatID == "" {
\t\tchatID = senderID
\t}
\tif senderID == "" {
\t\tsenderID = chatID
\t}
\tif chatID == "" || senderID == "" {
\t\tjsonError(w, "chatId or senderId required", 400)
\t\treturn
\t}

\tsenderName, _ := req["senderName"].(string)
\tif senderName == "" {
\t\tsenderName = "E2E"
\t}
\tchatName, _ := req["chatName"].(string)
\tif chatName == "" {
\t\tchatName = senderName
\t}
\tisGroup, _ := req["isGroup"].(bool)

\tmsgID, _ := req["messageId"].(string)
\tif msgID == "" {
\t\tmsgID = fmt.Sprintf("e2e-%d", time.Now().UnixNano())
\t}

\tmsg := map[string]interface{}{
\t\t"messageId":         msgID,
\t\t"chatId":            chatID,
\t\t"senderId":          senderID,
\t\t"senderName":        senderName,
\t\t"chatName":          chatName,
\t\t"isGroup":           isGroup,
\t\t"body":              body,
\t\t"hasMedia":          false,
\t\t"mediaType":         "",
\t\t"mediaUrls":         []string{},
\t\t"mentionedIds":      []string{},
\t\t"quotedMessageId":   "",
\t\t"quotedParticipant": "",
\t\t"quotedRemoteJid":   "",
\t\t"hasQuotedMessage":  false,
\t\t"botIds":            []string{},
\t\t"timestamp":         time.Now().Unix(),
\t}

\tqueueMu.Lock()
\tmessageQueue = append(messageQueue, msg)
\tqueueMu.Unlock()

\tlog.Printf("🧪 E2E inject from %s in %s: %s", senderID, chatID, truncate(body, 80))
\tjsonResponse(w, map[string]interface{}{"success": true, "messageId": msgID})
}
'''
    if '\t"net"\n' not in text:
        text = text.replace('"net/http"', '"net"\n\t"net/http"', 1)

    text = text.replace("func handleMessages(w http.ResponseWriter", handler + "\nfunc handleMessages(w http.ResponseWriter", 1)
    MAIN.write_text(text)
    print("patched", MAIN)


if __name__ == "__main__":
    main()
