package store

import (
    "crypto/rand"
    "encoding/hex"
)

// newID generates a prefixed, random identifier — e.g. "conv_9f2a...",
// "msg_1b4c..." — the same readable-prefix convention used by Stripe,
// Twilio, and GitHub's own REST APIs, so IDs are self-describing at a
// glance in logs, webhook payloads, and support conversations.
func newID(prefix string) string {
    b := make([]byte, 16)
    if _, err := rand.Read(b); err != nil {
        panic("store: crypto/rand unavailable: " + err.Error())
    }
    return prefix + "_" + hex.EncodeToString(b)
}
