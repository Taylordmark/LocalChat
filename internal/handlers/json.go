package handlers

import (
    "encoding/json"
    "net/http"

    "localchat/internal/store"
)

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, status int, msg string) {
    writeJSON(w, status, map[string]string{"error": msg})
}

func writeNotFoundOr500(w http.ResponseWriter, err error) {
    if err == store.ErrNotFound {
        writeError(w, http.StatusNotFound, "not found")
        return
    }
    writeError(w, http.StatusInternalServerError, err.Error())
}
