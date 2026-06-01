package http

import (
	"deployhub/internal/settings"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type settingsData struct {
	Settings  *settings.Settings
	Message   string
	Error     string
	CSRFToken string
	Username  string
}

type traefikExampleData struct {
	Hostname string
	YAML     string
}

func (s *Server) settingsPage(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.settings.Get()
	if err != nil {
		s.render.RenderError(w, r, 500, "Failed to load settings")
		return
	}

	username := s.auth.GetUsername(r)
	s.render.Render(w, r, "settings.html", settingsData{
		Settings:  cfg,
		CSRFToken: s.csrfToken(r),
		Username:  username,
	})
}

func (s *Server) settingsUpdate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.render.RenderError(w, r, 400, "Invalid form data")
		return
	}

	timeout, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("deploy_timeout")))
	if timeout <= 0 {
		timeout = 300
	}

	cfg := &settings.Settings{
		AppURL:              strings.TrimSpace(r.FormValue("app_url")),
		Timezone:            strings.TrimSpace(r.FormValue("timezone")),
		DockerSocket:        strings.TrimSpace(r.FormValue("docker_socket")),
		DeployCommand:       strings.TrimSpace(r.FormValue("deploy_command")),
		DeployTimeout:       timeout,
		WebhookSecret:       strings.TrimSpace(r.FormValue("webhook_secret")),
		TraefikDomain:       strings.TrimSpace(r.FormValue("traefik_domain")),
		TraefikEmail:        strings.TrimSpace(r.FormValue("traefik_email")),
		TraefikHTTPEntrypoint:  strings.TrimSpace(r.FormValue("traefik_http_entrypoint")),
		TraefikHTTPSEntrypoint: strings.TrimSpace(r.FormValue("traefik_https_entrypoint")),
		TraefikNetwork:      strings.TrimSpace(r.FormValue("traefik_network")),
	}

	if err := s.settings.Update(cfg); err != nil {
		username := s.auth.GetUsername(r)
		s.render.Render(w, r, "settings.html", settingsData{
			Settings:  cfg,
			Error:     "Failed to save settings: " + err.Error(),
			CSRFToken: s.csrfToken(r),
			Username:  username,
		})
		return
	}

	currentPass := r.FormValue("current_password")
	newPass := r.FormValue("new_password")
	if currentPass != "" && newPass != "" {
		if !s.auth.VerifyCurrentPassword(currentPass) {
			username := s.auth.GetUsername(r)
			s.render.Render(w, r, "settings.html", settingsData{
				Settings:  cfg,
				Error:     "Current password is incorrect",
				CSRFToken: s.csrfToken(r),
				Username:  username,
			})
			return
		}
		if len(newPass) < 8 {
			username := s.auth.GetUsername(r)
			s.render.Render(w, r, "settings.html", settingsData{
				Settings:  cfg,
				Error:     "New password must be at least 8 characters",
				CSRFToken: s.csrfToken(r),
				Username:  username,
			})
			return
		}
		if err := s.auth.UpdatePassword(newPass); err != nil {
			username := s.auth.GetUsername(r)
			s.render.Render(w, r, "settings.html", settingsData{
				Settings:  cfg,
				Error:     "Failed to update password: " + err.Error(),
				CSRFToken: s.csrfToken(r),
				Username:  username,
			})
			return
		}
	}

	username := s.auth.GetUsername(r)
	s.render.Render(w, r, "settings.html", settingsData{
		Settings:  cfg,
		Message:   "Settings saved successfully",
		CSRFToken: s.csrfToken(r),
		Username:  username,
	})
}

func (s *Server) settingsTraefikExample(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("hostname")
	if domain == "" {
		domain = "mydomain.com"
	}

	yaml := fmt.Sprintf(`services:
  deployhub:
    image: deployhub:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    env_file: .env
    volumes:
      - ./data:/data
      - /var/run/docker.sock:/var/run/docker.sock
    networks:
      - traefik
    labels:
      - traefik.enable=true
      - traefik.http.routers.deployhub.rule=Host(\` + "`" + domain + "`" + `)
      - traefik.http.routers.deployhub.entrypoints=websecure
      - traefik.http.routers.deployhub.tls=true
      - traefik.http.routers.deployhub.tls.certresolver=letsencrypt
      - traefik.http.services.deployhub.loadbalancer.server.port=8080

networks:
  traefik:
    external: true`)

	data := traefikExampleData{
		Hostname: domain,
		YAML:     yaml,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	s.render.templates.ExecuteTemplate(w, "partial_traefik_config.html", data)
}
