// ==========================================================
// CONVERSATIONS API CLIENT
// A thin UI over localchat's external Conversations API (/api/v1/...).
// Any system with an API key can create conversations, post messages, and
// bind delivery channels the same way this page does — this view is just
// one such client, not a special case of the API.
// ==========================================================

const apiBase = import.meta.env.VITE_API_BASE;
const KEY_STORAGE = "lc_conversations_api_key";

function getApiKey() {
    return localStorage.getItem(KEY_STORAGE) || "";
}
function setApiKey(key) {
    localStorage.setItem(KEY_STORAGE, key);
}

async function apiFetch(path, opts = {}) {
    const key = getApiKey();
    const headers = Object.assign({ "Content-Type": "application/json" }, opts.headers || {});
    if (key) headers["Authorization"] = `Bearer ${key}`;

    const res = await fetch(`${apiBase}/api/v1${path}`, Object.assign({}, opts, { headers }));
    if (res.status === 204) return null;

    let body = null;
    try {
        body = await res.json();
    } catch {
        // no body
    }
    if (!res.ok) {
        const msg = (body && body.error) || `HTTP ${res.status}`;
        const err = new Error(msg);
        err.status = res.status;
        throw err;
    }
    return body;
}

// ==========================================================
// VIEW SWITCHING
// ==========================================================
const mainEl = document.getElementById("main");
const convViewEl = document.getElementById("conversations-view");

document.getElementById("view-conversations").onclick = () => {
    mainEl.classList.add("hidden");
    convViewEl.classList.remove("hidden");
    refreshConversations();
    if (!getApiKey()) openSettings();
};

document.getElementById("view-assistant").onclick = () => {
    convViewEl.classList.add("hidden");
    mainEl.classList.remove("hidden");
};

// ==========================================================
// SETTINGS MODAL
// ==========================================================
const settingsModal = document.getElementById("conv-settings-modal");

function openSettings() {
    settingsModal.classList.remove("hidden");
    document.getElementById("conv-api-key-input").value = getApiKey();
    refreshWhoAmI();
    refreshKeys();
    refreshChannels();
}
function closeSettings() {
    settingsModal.classList.add("hidden");
}

document.getElementById("conv-settings-open").onclick = openSettings;
document.getElementById("conv-settings-close").onclick = closeSettings;
settingsModal.addEventListener("click", (e) => {
    if (e.target === settingsModal) closeSettings();
});

document.getElementById("conv-api-key-save").onclick = async () => {
    const val = document.getElementById("conv-api-key-input").value.trim();
    setApiKey(val);
    await refreshWhoAmI();
    refreshConversations();
    refreshKeys();
    refreshChannels();
};

async function refreshWhoAmI() {
    const el = document.getElementById("conv-whoami");
    if (!getApiKey()) {
        el.textContent = "No key saved yet.";
        return;
    }
    try {
        const key = await apiFetch("/whoami");
        el.textContent = `Authenticated as "${key.name}" (key ${key.id}).`;
    } catch (e) {
        el.textContent = `Key invalid: ${e.message}`;
    }
}

// ==========================================================
// API KEY MANAGEMENT
// ==========================================================
document.getElementById("conv-key-create").onclick = async () => {
    const nameInput = document.getElementById("conv-key-name");
    const name = nameInput.value.trim();
    if (!name) return;
    try {
        const created = await apiFetch("/keys", { method: "POST", body: JSON.stringify({ name }) });
        const box = document.getElementById("conv-new-key-secret");
        box.textContent = `New key "${created.name}" — copy it now, it will not be shown again:\n${created.secret}`;
        box.classList.remove("hidden");
        nameInput.value = "";
        refreshKeys();
    } catch (e) {
        alert("Failed to create key: " + e.message);
    }
};

async function refreshKeys() {
    const list = document.getElementById("conv-key-list");
    list.innerHTML = "";
    if (!getApiKey()) return;
    let keys = [];
    try {
        keys = await apiFetch("/keys");
    } catch {
        return;
    }
    keys.forEach((k) => {
        const li = document.createElement("li");
        const info = document.createElement("span");
        info.className = "settings-list-info";
        const used = k.lastUsedAt ? `last used ${new Date(k.lastUsedAt).toLocaleString()}` : "never used";
        info.textContent = `${k.name} — ${used}`;
        const del = document.createElement("button");
        del.textContent = "Revoke";
        del.onclick = async () => {
            if (!confirm(`Revoke key "${k.name}"?`)) return;
            await apiFetch(`/keys/${k.id}`, { method: "DELETE" });
            refreshKeys();
        };
        li.appendChild(info);
        li.appendChild(del);
        list.appendChild(li);
    });
}

// ==========================================================
// CHANNEL MANAGEMENT
// ==========================================================
document.getElementById("conv-channel-create").onclick = async () => {
    const url = document.getElementById("conv-channel-url").value.trim();
    const secret = document.getElementById("conv-channel-secret").value.trim();
    const scoped = document.getElementById("conv-channel-scoped").checked;
    if (!url) return;

    const body = { type: "webhook", config: { url } };
    if (secret) body.config.secret = secret;
    if (scoped && currentConversationId) body.conversationId = currentConversationId;

    try {
        await apiFetch("/channels", { method: "POST", body: JSON.stringify(body) });
        document.getElementById("conv-channel-url").value = "";
        document.getElementById("conv-channel-secret").value = "";
        refreshChannels();
    } catch (e) {
        alert("Failed to add channel: " + e.message);
    }
};

async function refreshChannels() {
    const list = document.getElementById("conv-channel-list");
    list.innerHTML = "";
    if (!getApiKey()) return;
    let channels = [];
    try {
        channels = await apiFetch("/channels");
    } catch {
        return;
    }
    channels.forEach((c) => {
        const li = document.createElement("li");
        const info = document.createElement("span");
        info.className = "settings-list-info";
        const scope = c.conversationId ? `conversation ${c.conversationId}` : "global";
        info.textContent = `${c.type}: ${c.config.url} (${scope})`;
        const del = document.createElement("button");
        del.textContent = "Remove";
        del.onclick = async () => {
            await apiFetch(`/channels/${c.id}`, { method: "DELETE" });
            refreshChannels();
        };
        li.appendChild(info);
        li.appendChild(del);
        list.appendChild(li);
    });
}

// ==========================================================
// CONVERSATIONS + MESSAGES
// ==========================================================
let currentConversationId = null;

document.getElementById("conv-new").onclick = async () => {
    const title = prompt("Conversation title:");
    if (title === null) return;
    try {
        const conv = await apiFetch("/conversations", { method: "POST", body: JSON.stringify({ title }) });
        await refreshConversations();
        selectConversation(conv.id);
    } catch (e) {
        alert("Failed to create conversation: " + e.message);
        if (e.status === 401) openSettings();
    }
};

async function refreshConversations() {
    const list = document.getElementById("conv-list");
    if (!getApiKey()) {
        list.innerHTML = '<div class="empty-state">Add an API key in Settings to load conversations.</div>';
        return;
    }
    let convs = [];
    try {
        convs = await apiFetch("/conversations");
    } catch (e) {
        list.innerHTML = `<div class="empty-state">Failed to load: ${e.message}</div>`;
        if (e.status === 401) openSettings();
        return;
    }

    list.innerHTML = "";
    if (convs.length === 0) {
        list.innerHTML = '<div class="empty-state">No conversations yet.</div>';
        return;
    }
    convs.forEach((c) => {
        const li = document.createElement("li");
        li.dataset.id = c.id;
        if (c.id === currentConversationId) li.classList.add("active");
        li.innerHTML = `
            <div class="conv-list-title">${c.title || "(untitled)"}</div>
            <div class="conv-list-meta">updated ${new Date(c.updatedAt).toLocaleString()}</div>
        `;
        li.onclick = () => selectConversation(c.id);
        list.appendChild(li);
    });
}

async function selectConversation(id) {
    currentConversationId = id;
    document.querySelectorAll("#conv-list li").forEach((li) => {
        li.classList.toggle("active", li.dataset.id === id);
    });

    const header = document.getElementById("conv-thread-header");
    const thread = document.getElementById("conv-thread");
    thread.innerHTML = '<div class="empty-state">Loading…</div>';

    try {
        const conv = await apiFetch(`/conversations/${id}`);
        header.textContent = `${conv.title || "(untitled)"} — ${conv.id}`;
        renderThread(conv.messages || []);
    } catch (e) {
        thread.innerHTML = `<div class="empty-state">Failed to load: ${e.message}</div>`;
    }
}

function renderThread(messages) {
    const thread = document.getElementById("conv-thread");
    thread.innerHTML = "";
    if (messages.length === 0) {
        thread.innerHTML = '<div class="empty-state">No messages yet.</div>';
        return;
    }
    messages.forEach((m) => {
        const div = document.createElement("div");
        div.className = "conv-message";
        const meta = document.createElement("div");
        meta.className = "conv-message-meta";
        meta.textContent = `${m.role}${m.author ? " · " + m.author : ""} · ${new Date(m.createdAt).toLocaleString()}`;
        const content = document.createElement("div");
        content.className = "conv-message-content";
        content.textContent = m.content;
        div.appendChild(meta);
        div.appendChild(content);
        thread.appendChild(div);
    });
    thread.scrollTop = thread.scrollHeight;
}

document.getElementById("conv-reply-send").onclick = async () => {
    if (!currentConversationId) {
        alert("Select or create a conversation first.");
        return;
    }
    const roleInput = document.getElementById("conv-reply-role");
    const contentInput = document.getElementById("conv-reply-content");
    const content = contentInput.value.trim();
    if (!content) return;
    const role = roleInput.value.trim() || "agent";

    try {
        await apiFetch(`/conversations/${currentConversationId}/messages`, {
            method: "POST",
            body: JSON.stringify({ role, content }),
        });
        contentInput.value = "";
        selectConversation(currentConversationId);
        refreshConversations();
    } catch (e) {
        alert("Failed to send: " + e.message);
    }
};
