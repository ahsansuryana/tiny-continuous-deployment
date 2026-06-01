package http

import (
	"deployhub/internal/auth"
	"deployhub/internal/deploy"
	"deployhub/internal/docker"
	"deployhub/internal/project"
	"deployhub/internal/settings"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Server struct {
	auth     *auth.Auth
	projects *project.Service
	deployer *deploy.Deployer
	docker   *docker.Client
	settings *settings.Service
	render   *Renderer
	assets   fs.FS

	rateLimitMu sync.Mutex
	rateLimit   map[string]*rateLimiter
}

type rateLimiter struct {
	count    int
	resetAt  time.Time
}

func New(
	au *auth.Auth,
	ps *project.Service,
	dp *deploy.Deployer,
	dc *docker.Client,
	ss *settings.Service,
	assets fs.FS,
) *Server {
	render, err := NewRenderer(assets, "web/templates/*.html")
	if err != nil {
		log.Fatalf("failed to parse templates: %v", err)
	}

	return &Server{
		auth:       au,
		projects:   ps,
		deployer:   dp,
		docker:     dc,
		settings:   ss,
		render:     render,
		assets:     assets,
		rateLimit:  make(map[string]*rateLimiter),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /login", s.loginPage)
	mux.HandleFunc("POST /login", s.login)
	mux.HandleFunc("POST /logout", s.logout)

	mux.HandleFunc("GET /", s.dashboard)
	mux.HandleFunc("GET /projects", s.projectList)
	mux.HandleFunc("GET /projects/new", s.projectNewForm)
	mux.HandleFunc("POST /projects/new", s.projectCreate)
	mux.HandleFunc("GET /projects/{id}", s.projectDetail)
	mux.HandleFunc("GET /projects/{id}/edit", s.projectEditForm)
	mux.HandleFunc("POST /projects/{id}/edit", s.projectUpdate)
	mux.HandleFunc("POST /projects/{id}/deploy", s.projectDeploy)
	mux.HandleFunc("POST /projects/{id}/stop", s.projectStop)
	mux.HandleFunc("GET /projects/{id}/logs", s.projectLogs)
	mux.HandleFunc("GET /projects/{id}/logs/stream", s.projectLogStream)
	mux.HandleFunc("GET /projects/{id}/container-logs", s.projectContainerLogs)
	mux.HandleFunc("GET /projects/{id}/container-logs/stream", s.projectContainerLogStream)
	mux.HandleFunc("POST /projects/{id}/delete", s.projectDelete)

	mux.HandleFunc("GET /settings", s.settingsPage)
	mux.HandleFunc("POST /settings", s.settingsUpdate)
	mux.HandleFunc("GET /settings/traefik", s.settingsTraefikExample)

	mux.HandleFunc("POST /api/deploy/{project}", s.webhookDeploy)

	staticFS, err := fs.Sub(s.assets, "web/static")
	if err == nil {
		mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	}

	var handler http.Handler = mux

	handler = s.auth.CSRFMiddleware(handler)
	handler = s.auth.Middleware(handler)
	handler = s.rateLimitMiddleware(handler)
	handler = s.sessionMiddleware(handler)

	return handler
}

func (s *Server) sessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		ip := r.RemoteAddr
		if idx := strings.LastIndex(ip, ":"); idx >= 0 {
			ip = ip[:idx]
		}

		s.rateLimitMu.Lock()
		rl, ok := s.rateLimit[ip]
		now := time.Now()

		if !ok || now.After(rl.resetAt) {
			rl = &rateLimiter{count: 0, resetAt: now.Add(time.Minute)}
			s.rateLimit[ip] = rl
		}

		rl.count++
		remaining := 100 - rl.count
		s.rateLimitMu.Unlock()

		w.Header().Set("X-RateLimit-Limit", "100")
		w.Header().Set("X-RateLimit-Remaining", itoa(remaining))

		if rl.count > 100 {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) csrfToken(r *http.Request) string {
	return s.auth.EnsureCSRFToken(r)
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
