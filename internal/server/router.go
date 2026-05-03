func NewRouter(cfg config.Config) http.Handler {
    r := chi.NewRouter()

    // 1. Middlewares first
    r.Use(middleware.Recovery)
    r.Use(middleware.Logging)
    r.Use(middleware.CORS)

    // 2. Specific API Routes
    r.Get("/health", handlers.Health)
    r.Post("/api/chat", handlers.Chat(cfg))

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