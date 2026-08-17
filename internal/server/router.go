package server

import (
    "net/http"
    "os"
    "path/filepath"
    "strings"

    "github.com/go-chi/chi/v5"

    "localchat/internal/apiauth"
    "localchat/internal/config"
    "localchat/internal/handlers"
    "localchat/internal/middleware"
)


// NewRouter wires the two independent surfaces this server exposes:
//   - the original personal chat UI's endpoints (/api/chat, /api/models,
//     /health, and the static web/ assets) — unauthenticated, local-only,
//     unchanged from before
//   - the external Conversations API (/api/v1/...) — API-key gated, meant
//     for any external system to integrate against
func NewRouter(cfg config.Config, api *handlers.ConversationsAPI) http.Handler {
    r := chi.NewRouter()

    // 1. Middlewares first
    r.Use(middleware.Recovery)
    r.Use(middleware.Logging)
    r.Use(middleware.CORS)

    // 2. Specific API Routes
    r.Get("/health", handlers.Health)
    r.Post("/api/chat", handlers.Chat(cfg))
    r.Get("/api/models", handlers.ListModels(cfg))

    // 2b. Conversations API — API-key gated, see internal/apiauth.
    r.Route("/api/v1", func(v1 chi.Router) {
        v1.Use(apiauth.Middleware(api.Store))

        v1.Get("/whoami", api.WhoAmI)

        v1.Route("/conversations", func(cr chi.Router) {
            cr.Post("/", api.CreateConversation)
            cr.Get("/", api.ListConversations)
            cr.Route("/{id}", func(cir chi.Router) {
                cir.Get("/", api.GetConversation)
                cir.Route("/messages", func(mr chi.Router) {
                    mr.Post("/", api.PostMessage)
                    mr.Get("/", api.ListMessages)
                })
            })
        })

        v1.Route("/channels", func(cr chi.Router) {
            cr.Post("/", api.CreateChannel)
            cr.Get("/", api.ListChannels)
            cr.Delete("/{id}", api.DeleteChannel)
        })

        v1.Route("/keys", func(cr chi.Router) {
            cr.Post("/", api.CreateKey)
            cr.Get("/", api.ListKeys)
            cr.Delete("/{id}", api.RevokeKey)
        })
    })


    // 3. Static File Serving
    // We use a helper to ensure Chi handles the sub-pathing correctly
    workDir, _ := os.Getwd()
    filesDir := http.Dir(filepath.Join(workDir, "web"))
    
    // This is the standard Chi way to serve a directory
    FileServer(r, "/", filesDir)

    return r
}

// Add this helper function at the bottom of the file (or in a utils file)
// it handles stripping the prefix so the server looks for 'index.html' 
// inside 'web/' instead of 'web/index.html'
func FileServer(r chi.Router, path string, root http.FileSystem) {
    if strings.ContainsAny(path, "{}*") {
        panic("FileServer does not permit any URL parameters.")
    }

    if path != "/" && path[len(path)-1] != '/' {
        r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
        path += "/"
    }
    path += "*"

    r.Get(path, func(w http.ResponseWriter, r *http.Request) {
        rctx := chi.RouteContext(r.Context())
        pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
        fs := http.StripPrefix(pathPrefix, http.FileServer(root))
        fs.ServeHTTP(w, r)
    })
}