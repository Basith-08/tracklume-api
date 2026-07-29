package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Basith-08/tracklume-api/internal/auth"
	"github.com/Basith-08/tracklume-api/internal/config"
	"github.com/Basith-08/tracklume-api/internal/dashboard"
	"github.com/Basith-08/tracklume-api/internal/issue"
	"github.com/Basith-08/tracklume-api/internal/middleware"
	"github.com/Basith-08/tracklume-api/internal/project"
	"github.com/Basith-08/tracklume-api/internal/response"
	"github.com/Basith-08/tracklume-api/internal/security"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(cfg config.Config, pool *pgxpool.Pool, logger *slog.Logger) http.Handler {
	tokens := security.NewTokenManager(cfg.JWTSecret, cfg.JWTExpiration)
	authHandler := auth.NewHandler(auth.NewService(auth.NewRepository(pool), tokens))
	projectService := project.NewService(project.NewRepository(pool))
	projectHandler := project.NewHandler(projectService)
	issueHandler := issue.NewHandler(issue.NewService(issue.NewRepository(pool), projectService, pool))
	dashboardHandler := dashboard.NewHandler(dashboard.NewService(dashboard.NewRepository(pool), projectService))
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.SecurityHeaders, middleware.CORS(cfg.CORSOrigins), middleware.Recover(logger), middleware.Logging(logger), middleware.BodyLimit(cfg.BodyLimit), func(next http.Handler) http.Handler { return middleware.Timeout(cfg.RequestTimeout, next) })
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		response.Write(w, 200, map[string]string{"status": "ok", "service": "tracklume-api"})
	})
	r.Get("/openapi.yaml", openAPI)
	r.Get("/docs", swaggerUI)
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			response.WriteError(w, r, 503, "NOT_READY", "Database is not ready", nil)
			return
		}
		response.Write(w, 200, map[string]string{"status": "ready"})
	})
	r.Route("/api/v1", func(api chi.Router) {
		api.Route("/auth", func(r chi.Router) {
			r.Use(middleware.RateLimit(cfg.RateLimitRequests, cfg.RateLimitWindow))
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
		})
		api.Group(func(secured chi.Router) {
			secured.Use(func(next http.Handler) http.Handler { return middleware.Authenticate(tokens, next) })
			secured.Get("/me", authHandler.Me)
			secured.Patch("/me", authHandler.UpdateMe)
			secured.Put("/me/password", authHandler.ChangePassword)
			secured.Get("/projects", projectHandler.List)
			secured.Post("/projects", projectHandler.Create)
			secured.Get("/projects/{projectID}", projectHandler.Get)
			secured.Patch("/projects/{projectID}", projectHandler.Update)
			secured.Delete("/projects/{projectID}", projectHandler.Delete)
			secured.Get("/projects/{projectID}/issues/{issueID}", issueHandler.Get)
			secured.Patch("/projects/{projectID}/issues/{issueID}", issueHandler.Update)
			secured.Delete("/projects/{projectID}/issues/{issueID}", issueHandler.Delete)
			secured.Route("/projects/{projectID}", func(p chi.Router) {
				p.Get("/", projectHandler.Get)
				p.Patch("/", projectHandler.Update)
				p.Delete("/", projectHandler.Delete)
				p.Get("/members", projectHandler.Members)
				p.Post("/members", projectHandler.AddMember)
				p.Patch("/members/{userID}", projectHandler.UpdateMember)
				p.Delete("/members/{userID}", projectHandler.RemoveMember)
				p.Get("/dashboard", dashboardHandler.Get)
				p.Get("/issues", issueHandler.List)
				p.Post("/issues", issueHandler.Create)
				p.Route("/issues/{issueID}", func(i chi.Router) {
					i.Get("/", issueHandler.Get)
					i.Patch("/", issueHandler.Update)
					i.Delete("/", issueHandler.Delete)
					i.Patch("/status", issueHandler.Status)
					i.Patch("/position", issueHandler.Position)
					i.Get("/activities", issueHandler.Activities)
				})
			})
		})
	})
	return r
}

func openAPI(w http.ResponseWriter, r *http.Request) {
	spec, err := os.ReadFile("openapi.yaml")
	if err != nil {
		response.WriteError(w, r, http.StatusNotFound, "DOCS_NOT_FOUND", "OpenAPI specification is not available", nil)
		return
	}
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(spec)
}

func swaggerUI(w http.ResponseWriter, _ *http.Request) {
	const page = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Tracklume API Documentation</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = () => SwaggerUIBundle({
      url: '/openapi.yaml',
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
      layout: 'BaseLayout'
    });
  </script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(page))
}
