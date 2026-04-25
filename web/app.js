// ==========================================================
// GLOBAL TTS CONTROLS
// ==========================================================
let currentUtterance = null;
let isSpeaking = false;

function speak(text) {
    stopSpeaking();
    currentUtterance = new SpeechSynthesisUtterance(text);
    currentUtterance.rate = 1.0;
    currentUtterance.pitch = 1.0;

    currentUtterance.onend = () => {
        isSpeaking = false;
    };

    speechSynthesis.speak(currentUtterance);
    isSpeaking = true;
}

function stopSpeaking() {
    if (speechSynthesis.speaking) {
        speechSynthesis.cancel();
    }
    isSpeaking = false;
}

function toggleSpeak(text, button) {
    if (!isSpeaking) {
        speak(text);
        button.textContent = "⏸ Pause";
    } else {
        stopSpeaking();
        button.textContent = "▶️ Play";
    }
}


// ==========================================================
// TYPING INDICATOR HELPERS
// ==========================================================
function showTyping() {
    const t = document.getElementById("typing-indicator");
    if (t) t.classList.remove("hidden");
}

function hideTyping() {
    const t = document.getElementById("typing-indicator");
    if (t) t.classList.add("hidden");
}


// ==========================================================
// MARKDOWN RENDERING (minimal but useful)
// ==========================================================
function escapeHtml(str) {
    return str
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;");
}

function renderMarkdown(text) {
    let html = escapeHtml(text);

    // Code blocks ```...```
    html = html.replace(/```([\s\S]*?)```/g, (m, code) => {
        return `<pre><code>${code.trim()}</code></pre>`;
    });

    // Inline code `...`
    html = html.replace(/`([^`]+)`/g, (m, code) => {
        return `<code>${code}</code>`;
    });

    // Bold **...**
    html = html.replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>");

    // Italic *...*
    html = html.replace(/\*(.+?)\*/g, "<em>$1</em>");

    // Simple bullet lists: "- "
    html = html.replace(/(^|\n)- (.+)/g, "$1• $2");

    // Line breaks
    html = html.replace(/\n/g, "<br>");

    return html;
}


// ==========================================================
// CONVERSATION STATE MANAGEMENT
// ==========================================================
let conversations = JSON.parse(localStorage.getItem("conversations") || "{}");
let currentChatId = null;

function createNewChat() {
    currentChatId = "chat_" + Date.now();
    conversations[currentChatId] = [];
    saveConversations();
    renderHistory();
    clearChatUI();
    hideTyping();
    setActionButtonsEnabled(true);
    highlightActiveInHistory();
}

function saveMessage(role, text, model = null) {
    if (!currentChatId) createNewChat();
    conversations[currentChatId].push({ role, text, model });
    saveConversations();
}

function saveConversations() {
    localStorage.setItem("conversations", JSON.stringify(conversations));
    // Hook for future server-side sync:
    // fetch("/api/conversations", {
    //   method: "POST",
    //   headers: { "Content-Type": "application/json" },
    //   body: JSON.stringify(conversations)
    // }).catch(() => {});
}

function loadChat(id) {
    currentChatId = id;
    clearChatUI();

    const msgs = conversations[id] || [];
    msgs.forEach(msg => {
        addMessageToUI(msg.role, msg.text, false, msg.model);
    });

    hideTyping();
    setActionButtonsEnabled(true);
    highlightActiveInHistory();
}

function renderHistory() {
    const history = document.getElementById("history");
    history.innerHTML = "";

    Object.keys(conversations)
        .sort((a, b) => b.localeCompare(a))
        .forEach(id => {
            const li = document.createElement("li");
            const first = conversations[id][0]?.text || "New conversation";
            li.innerHTML = `
                <div class="history-title">${first.slice(0, 40)}${first.length > 40 ? "…" : ""}</div>
                <div class="history-model">${conversations[id][0]?.model || ""}</div>
            `;

            li.dataset.id = id;
            li.onclick = () => {
                stopSpeaking();
                loadChat(id);
            };
            history.appendChild(li);
        });

    highlightActiveInHistory();
}

function highlightActiveInHistory() {
    const items = document.querySelectorAll("#history li");
    items.forEach(li => li.classList.remove("active"));
    if (!currentChatId) return;
    const active = [...items].find(li => li.dataset.id === currentChatId);
    if (active) active.classList.add("active");
}

function clearChatUI() {
    document.getElementById("chat").innerHTML = "";
}

function setActionButtonsEnabled(enabled) {
    const exp = document.getElementById("export-chat");
    const del = document.getElementById("delete-chat");
    if (exp) exp.disabled = !enabled;
    if (del) del.disabled = !enabled;
}


// ==========================================================
// EXPORT / DELETE CURRENT CHAT
// ==========================================================
function exportCurrentChat() {
    if (!currentChatId || !conversations[currentChatId]) return;

    const msgs = conversations[currentChatId];
    let text = "";

    msgs.forEach(m => {
        text += `${m.role.toUpperCase()}: ${m.text}\n\n`;
    });

    const blob = new Blob([text], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = currentChatId + ".txt";
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
}

function deleteCurrentChat() {
    if (!currentChatId) return;
    delete conversations[currentChatId];
    saveConversations();
    renderHistory();

    const keys = Object.keys(conversations);
    if (keys.length === 0) {
        currentChatId = null;
        clearChatUI();
        hideTyping();
        setActionButtonsEnabled(false);
    } else {
        const latest = keys.sort().pop();
        loadChat(latest);
    }
}


// ==========================================================
// MESSAGE UI HANDLER (with model badges + markdown)
// ==========================================================
function addMessageToUI(role, text, save = true, model = null) {
    const chat = document.getElementById("chat");

    const div = document.createElement("div");
    div.className = "message " + role;

    if (role === "bot") {
        const content = document.createElement("div");
        content.className = "message-content";
        content.innerHTML = renderMarkdown(text);

        const meta = document.createElement("div");
        meta.className = "message-meta";
        //if (model) {
        //    const badge = document.createElement("span");
        //    badge.className = "model-badge";
        //    badge.textContent = model;
        //    meta.appendChild(badge);
        //}

        div.appendChild(content);
        div.appendChild(meta);
    } else {
        div.textContent = text;
    }

    chat.appendChild(div);
    chat.scrollTop = chat.scrollHeight;

    if (save) saveMessage(role, text, model);
}


// ==========================================================
// MAIN CHAT LOGIC
// ==========================================================
document.getElementById("send").onclick = async () => {
    const model = document.getElementById("model").value;
    const input = document.getElementById("input");
    const prompt = input.value.trim();
    if (!prompt) return;

    addMessageToUI("user", prompt);

    input.value = "";
    input.blur();
    input.focus();

    showTyping();

    const res = await fetch("/api/chat", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ model, prompt })
    });

    const reader = res.body.getReader();
    const decoder = new TextDecoder();

    let botMessage = "";
    const chat = document.getElementById("chat");

    let botDiv = document.createElement("div");
    botDiv.className = "message bot";

    const content = document.createElement("div");
    content.className = "message-content";
    botDiv.appendChild(content);

    const meta = document.createElement("div");
    meta.className = "message-meta";
    //const badge = document.createElement("span");
    //badge.className = "model-badge";
    //badge.textContent = model;
    //meta.appendChild(badge);
    //botDiv.appendChild(meta);

    chat.appendChild(botDiv);

    try {
        while (true) {
            const { value, done } = await reader.read();
            if (done) break;

            const chunk = decoder.decode(value);
            const lines = chunk.split("\n").filter(l => l.trim() !== "");

            for (const line of lines) {
                try {
                    const json = JSON.parse(line);

                    if (json.response) {
                        botMessage += json.response;
                        content.innerHTML = renderMarkdown(botMessage);
                        chat.scrollTop = chat.scrollHeight;
                    }

                    if (json.done === true) {
                        hideTyping();

                        const auto = document.getElementById("tts-auto").checked;

                        if (auto) {
                            speak(botMessage);
                        } else {
                            const playBtn = document.createElement("button");
                            playBtn.textContent = "▶️ Play";
                            playBtn.className = "tts-play";
                            playBtn.onclick = () => toggleSpeak(botMessage, playBtn);
                            meta.appendChild(playBtn);
                        }

                        saveMessage("bot", botMessage, model);
                    }

                } catch (e) {
                    console.error("Bad JSON chunk:", line);
                }
            }
        }
    } finally {
        hideTyping();
    }
};


// ==========================================================
// NEW CHAT / EXPORT / DELETE BUTTONS
// ==========================================================
document.getElementById("new-chat").onclick = () => {
    stopSpeaking();
    createNewChat();
};

document.getElementById("export-chat").onclick = () => {
    stopSpeaking();
    exportCurrentChat();
};

document.getElementById("delete-chat").onclick = () => {
    stopSpeaking();
    deleteCurrentChat();
};


// ==========================================================
// THEME TOGGLE
// ==========================================================
document.getElementById("theme-toggle").onclick = () => {
    document.body.classList.toggle("light");
};


// ==========================================================
// INITIALIZE ON PAGE LOAD
// ==========================================================
window.onload = () => {
    hideTyping();
    renderHistory();

    const keys = Object.keys(conversations);
    if (keys.length === 0) {
        setActionButtonsEnabled(false);
        createNewChat();
    } else {
        setActionButtonsEnabled(true);
        const latest = keys.sort().pop();
        loadChat(latest);
    }
};
