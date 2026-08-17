# LocalChat Deployment Summary (Stateful Ollama Chat + Modern UI)

This project implements a fully local, stateful chat interface backed by Ollama’s `/api/chat` endpoint, with a responsive UI, collapsible sidebar, and streaming token updates. Below is a complete summary of the architecture, changes made, and how to extend the system safely.

---

# 1. Frontend Architecture

### Stateful message history  
The frontend maintains a full conversation array:

```js
let chatHistory = [];
```

Each send:

- Pushes `{role:"user", content:"..."}`  
- Sends the entire `messages` array to the backend  
- Streams assistant tokens  
- Pushes `{role:"assistant", content:"..."}`  

### Updated streaming parser  
Ollama streams:

```json
{"message":{"role":"assistant","content":"..."}, "done":false}
```

Frontend now reads:

```js
if (json.message && json.message.content) {
    botMessage += json.message.content;
}
```

### Mobile‑friendly input bar  
Input bar is fixed at the bottom on mobile using `position: fixed`.

### Collapsible sidebar  
A toggle button adds or removes the `.open` class on mobile.

### Full‑height layout  
The main container uses `100dvh` and flexbox to ensure the chat area always fits the viewport.

---

# 2. Backend Architecture

### Removed stateless `/api/generate`  
All prompt‑based code was removed.

### Backend now accepts:

```json
{
  "model": "llama3:8b",
  "messages": [
    {"role":"user","content":"..."},
    {"role":"assistant","content":"..."}
  ]
}
```

### Updated handler (`chat.go`)  
The handler:

1. Decodes `messages`
2. Calls `ollama.ChatStream`
3. Streams JSON lines back to the frontend

### Updated Ollama client (`client.go`)  
Only one function remains:

```go
func ChatStream(host, model string, messages []Message) (io.ReadCloser, error)
```

This POSTs to `/api/chat` and returns the streaming body.

---

# 3. UI and Layout Improvements

- Sidebar always on the left on desktop  
- Sidebar collapsible on mobile  
- Main content fills full height  
- Chat area expands dynamically  
- Input bar fixed at bottom  
- Uses `100dvh` to avoid mobile viewport issues  
- No layout shift between desktop and mobile  

---

# 4. How to Extend the System

## A. Adding new chat features (temperature, system prompts, etc.)

1. Add fields to the frontend request:

```js
body: JSON.stringify({
    model,
    messages: chatHistory,
    temperature,
    systemPrompt
})
```

2. Add matching fields to:

- `ChatRequest` in `chat.go`
- `ChatRequest` in `client.go`

3. Forward them to Ollama.

## B. Adding new UI components  
Safe areas to modify:

- `#controls`  
- `#topbar`  
- `#sidebar`  
- `#input-bar`  

Avoid modifying:

- `#chat` height logic  
- `#main` flex layout  
- `#input-bar` positioning  

## C. Adding new backend endpoints  
Follow the existing pattern:

1. Create a handler in `handlers/`  
2. Add a matching function in `ollama/`  
3. Register the route  
4. Use streaming with `io.ReadCloser` when needed  

## D. Adding persistent chat history  
Store conversations as:

```json
{id, title, messages}
```

Load messages into `chatHistory` when selected.

## E. Adding multi‑conversation support  
Use the sidebar to switch between stored conversations.  
Each conversation maintains its own `messages` array.

---

# 4a. Conversations API (external integrations)

Alongside the personal chat UI, the server exposes a second, independent surface: a generic **Conversations API** at `/api/v1/...`, in the same family as Twilio Conversations, Intercom, or Front. Any external system — a script, a backend service, an automation platform — can use it to open a conversation with a human, post messages into it, and get notified the moment a reply lands, all through plain HTTP. It has no knowledge of, or dependency on, any particular caller; the web UI's own "Conversations" tab is just one client of it.

## Concepts

- **Conversation** — a thread of messages. Has a `title` and a free-form `metadata` object a caller can use to stash its own correlation data (an external record ID, a category, anything) — it round-trips back out on every webhook delivery for that conversation.
- **Message** — one entry in a conversation. `role` and `author` are free text, not a fixed enum — express whatever participants make sense for the integration (`"customer"`, a phone number as `author`, `"system"`, etc.).
- **Channel** — a delivery destination bound to a conversation (or, if unscoped, to every conversation). Every new message fires every applicable channel. Today's built-in implementation is `webhook`: POST the message as JSON to a URL, HMAC-SHA256-signed with a per-channel secret in the `X-LocalChat-Signature` header (the same signing convention as Stripe/GitHub/Twilio webhooks) when a secret is configured.
- **API key** — bearer credential for this API (`Authorization: Bearer <key>` or `X-API-Key: <key>`). The server mints one bootstrap admin key on first run and prints it once to the log; use it to mint additional named keys per integration, and revoke any key without affecting the others.

## Endpoints

| Method | Path | Description |
|---|---|---|
| POST | `/api/v1/conversations` | Create a conversation. Optional `message` and `channel` fields create the opening message and bind a channel in the same call. |
| GET | `/api/v1/conversations` | List conversations, most recently updated first. |
| GET | `/api/v1/conversations/{id}` | Get a conversation with its full message history. |
| POST | `/api/v1/conversations/{id}/messages` | Post a message into a conversation — fires every bound channel. |
| GET | `/api/v1/conversations/{id}/messages` | List a conversation's messages. |
| POST | `/api/v1/channels` | Bind a channel. Omit `conversationId` for a global channel. |
| GET | `/api/v1/channels` | List all channels. |
| DELETE | `/api/v1/channels/{id}` | Remove a channel. |
| POST | `/api/v1/keys` | Mint a named API key — the secret is returned once, in this response only. |
| GET | `/api/v1/keys` | List keys (names and usage, never secrets). |
| DELETE | `/api/v1/keys/{id}` | Revoke a key. |
| GET | `/api/v1/whoami` | Identify the key that authenticated the request. |

## Example: open a conversation and get notified on reply

```bash
curl -X POST http://localhost:8080/api/v1/conversations \
  -H "Authorization: Bearer lc_..." -H "Content-Type: application/json" \
  -d '{
    "title": "Support request #482",
    "metadata": {"externalId": "482"},
    "message": {"role": "system", "content": "A customer needs help — please respond."},
    "channel": {"type": "webhook", "config": {"url": "https://your-service/hook", "secret": "shh"}}
  }'
```

Whenever a message is posted back into that conversation (by a human replying in the web UI's Conversations tab, or by another API call), `https://your-service/hook` receives a signed `POST`:

```json
{
  "event": "message.created",
  "conversation": {"id": "conv_...", "title": "Support request #482"},
  "message": {"id": "msg_...", "role": "customer", "content": "...", "createdAt": "..."},
  "metadata": {"externalId": "482"}
}
```

## Adding a new channel type

Implement the `Channel` interface in `internal/channel/` (`Type() string`, `Deliver(ctx, conversation, message, config) error`) and register it in `cmd/server/main.go`'s `channel.NewDispatcher(db, channel.NewWebhook(), yourChannel)` call. Nothing else — the store, the HTTP API, and the dispatcher are all channel-agnostic by design. This is the extension point for SMS, email, or any other delivery mechanism.

## Storage

SQLite, a single file at `LOCALCHAT_DB_PATH` (default `./localchat.db`; the Docker Compose setup mounts a named volume at `/data`). Pure-Go driver — no cgo, no separate database server, consistent with this project staying a single self-contained binary.

---

# 5. Deployment Notes

- Backend is a simple Go server  
- Frontend is static HTML/CSS/JS  
- Ollama must be running locally  
- No external dependencies  
- No cloud calls  

This is a fully offline, self‑contained LLM chat system.

---

# 6. Core Principles to Maintain

1. Frontend always sends full `messages` array  
2. Backend always forwards messages to `/api/chat`  
3. Streaming remains line‑based JSON  
4. UI layout remains flex‑based  
5. Mobile uses `100dvh` and `min-height: 0`  
6. Sidebar stays outside the main flex column  

---