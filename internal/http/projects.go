package http

import (
	"context"
	"deployhub/internal/project"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type projectListData struct {
	Projects  []project.ProjectStatus
	CSRFToken string
	Username  string
}

type projectFormData struct {
	Project   *project.Project
	Errors    map[string]string
	CSRFToken string
	Username  string
}

type projectDetailData struct {
	Project      *project.Project
	Status       string
	LastDeploy   string
	DeployID     int64
	TraefikDomain string
	CSRFToken    string
	Username     string
}

func (s *Server) projectList(w http.ResponseWriter, r *http.Request) {
	projects, err := s.projects.ListWithStatus()
	if err != nil {
		s.render.RenderError(w, r, 500, "Failed to load projects")
		return
	}

	username := s.auth.GetUsername(r)
	s.render.Render(w, r, "project_list.html", projectListData{
		Projects:  projects,
		CSRFToken: s.csrfToken(r),
		Username:  username,
	})
}

func (s *Server) projectNewForm(w http.ResponseWriter, r *http.Request) {
	username := s.auth.GetUsername(r)
	s.render.Render(w, r, "project_form.html", projectFormData{
		Project:   &project.Project{AutoDeploy: true},
		Errors:    nil,
		CSRFToken: s.csrfToken(r),
		Username:  username,
	})
}

func (s *Server) projectCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.render.RenderError(w, r, 400, "Invalid form data")
		return
	}

	traefikPort, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("traefik_port")))

	p := &project.Project{
		Name:             strings.TrimSpace(r.FormValue("name")),
		RepoURL:          strings.TrimSpace(r.FormValue("repo_url")),
		ComposePath:      strings.TrimSpace(r.FormValue("compose_path")),
		ComposeType:      strings.TrimSpace(r.FormValue("compose_type")),
		ComposeContent:   strings.TrimSpace(r.FormValue("compose_content")),
		WebhookToken:     strings.TrimSpace(r.FormValue("webhook_token")),
		Branch:           strings.TrimSpace(r.FormValue("branch")),
		RegistryType:     strings.TrimSpace(r.FormValue("registry_type")),
		RegistryUser:     strings.TrimSpace(r.FormValue("registry_username")),
		RegistryPass:     strings.TrimSpace(r.FormValue("registry_password")),
		AutoDeploy:       r.FormValue("auto_deploy") == "on",
		TraefikHostname:  strings.TrimSpace(r.FormValue("traefik_hostname")),
		TraefikPort:      traefikPort,
		TraefikMiddleware: strings.TrimSpace(r.FormValue("traefik_middleware")),
	}

	if errMsg := p.Validate(); errMsg != "" {
		username := s.auth.GetUsername(r)
		s.render.Render(w, r, "project_form.html", projectFormData{
			Project:   p,
			Errors:    map[string]string{"name": errMsg},
			CSRFToken: s.csrfToken(r),
			Username:  username,
		})
		return
	}

	if err := s.projects.Create(p); err != nil {
		username := s.auth.GetUsername(r)
		errMsg := err.Error()
		if strings.Contains(errMsg, "UNIQUE") {
			errMsg = "A project with this name already exists"
		}
		s.render.Render(w, r, "project_form.html", projectFormData{
			Project:   p,
			Errors:    map[string]string{"name": errMsg},
			CSRFToken: s.csrfToken(r),
			Username:  username,
		})
		return
	}

	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}

func (s *Server) projectDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.render.RenderError(w, r, 400, "Invalid project ID")
		return
	}

	p, err := s.projects.GetByID(id)
	if err != nil {
		s.render.RenderError(w, r, 404, "Project not found")
		return
	}

	statuses, err := s.projects.ListWithStatus()
	var status string
	var lastDeploy string
	var deployID int64
	if err == nil {
		for _, ps := range statuses {
			if ps.ID == id {
				status = ps.Status
				lastDeploy = ps.LastDeployAt
				deployID = ps.DeploymentID
				break
			}
		}
	}

	username := s.auth.GetUsername(r)
	traefikDomain, _ := s.settings.Get()
	s.render.Render(w, r, "project_detail.html", projectDetailData{
		Project:       p,
		Status:        status,
		LastDeploy:    lastDeploy,
		DeployID:      deployID,
		TraefikDomain: traefikDomain.TraefikDomain,
		CSRFToken:     s.csrfToken(r),
		Username:      username,
	})
}

func (s *Server) projectEditForm(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.render.RenderError(w, r, 400, "Invalid project ID")
		return
	}

	p, err := s.projects.GetByID(id)
	if err != nil {
		s.render.RenderError(w, r, 404, "Project not found")
		return
	}

	username := s.auth.GetUsername(r)
	s.render.Render(w, r, "project_form.html", projectFormData{
		Project:   p,
		Errors:    nil,
		CSRFToken: s.csrfToken(r),
		Username:  username,
	})
}

func (s *Server) projectUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.render.RenderError(w, r, 400, "Invalid project ID")
		return
	}

	if err := r.ParseForm(); err != nil {
		s.render.RenderError(w, r, 400, "Invalid form data")
		return
	}

	traefikPort, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("traefik_port")))

	p := &project.Project{
		ID:               id,
		Name:             strings.TrimSpace(r.FormValue("name")),
		RepoURL:          strings.TrimSpace(r.FormValue("repo_url")),
		ComposePath:      strings.TrimSpace(r.FormValue("compose_path")),
		ComposeType:      strings.TrimSpace(r.FormValue("compose_type")),
		ComposeContent:   strings.TrimSpace(r.FormValue("compose_content")),
		WebhookToken:     strings.TrimSpace(r.FormValue("webhook_token")),
		Branch:           strings.TrimSpace(r.FormValue("branch")),
		RegistryType:     strings.TrimSpace(r.FormValue("registry_type")),
		RegistryUser:     strings.TrimSpace(r.FormValue("registry_username")),
		RegistryPass:     strings.TrimSpace(r.FormValue("registry_password")),
		AutoDeploy:       r.FormValue("auto_deploy") == "on",
		TraefikHostname:  strings.TrimSpace(r.FormValue("traefik_hostname")),
		TraefikPort:      traefikPort,
		TraefikMiddleware: strings.TrimSpace(r.FormValue("traefik_middleware")),
	}

	if errMsg := p.Validate(); errMsg != "" {
		username := s.auth.GetUsername(r)
		s.render.Render(w, r, "project_form.html", projectFormData{
			Project:   p,
			Errors:    map[string]string{"name": errMsg},
			CSRFToken: s.csrfToken(r),
			Username:  username,
		})
		return
	}

	if err := s.projects.Update(p); err != nil {
		username := s.auth.GetUsername(r)
		s.render.Render(w, r, "project_form.html", projectFormData{
			Project:   p,
			Errors:    map[string]string{"name": err.Error()},
			CSRFToken: s.csrfToken(r),
			Username:  username,
		})
		return
	}

	http.Redirect(w, r, "/projects/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

func (s *Server) projectStop(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.render.RenderError(w, r, 400, "Invalid project ID")
		return
	}

	p, err := s.projects.GetByID(id)
	if err != nil {
		s.render.RenderError(w, r, 404, "Project not found")
		return
	}

	go func() {
		err := s.docker.StreamCommand(context.Background(), p.DeployDir(), "docker compose stop", make(chan string, 1))
		if err != nil {
			log.Printf("failed to stop project %d: %v", id, err)
		}
	}()

	http.Redirect(w, r, "/projects/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

func (s *Server) projectContainerLogs(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.render.RenderError(w, r, 400, "Invalid project ID")
		return
	}

	p, err := s.projects.GetByID(id)
	if err != nil {
		s.render.RenderError(w, r, 404, "Project not found")
		return
	}

	username := s.auth.GetUsername(r)
	s.render.Render(w, r, "log_viewer_container.html", logViewerData{
		ProjectName: p.Name,
		ProjectID:   id,
		CSRFToken:   s.csrfToken(r),
		Username:    username,
	})
}

func (s *Server) projectContainerLogStream(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	p, err := s.projects.GetByID(id)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ctx := r.Context()
	lines := make(chan string, 100)

	go func() {
		defer close(lines)
		s.docker.StreamCommand(ctx, p.DeployDir(), "docker compose logs --tail=100 --no-color -f", lines)
	}()

	for line := range lines {
		select {
		case <-ctx.Done():
			return
		default:
			fmt.Fprintf(w, "data: %s\n\n", escapeHTML(line))
			flusher.Flush()
		}
	}
}

func (s *Server) projectDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.render.RenderError(w, r, 400, "Invalid project ID")
		return
	}

	p, err := s.projects.GetByID(id)
	if err != nil {
		s.render.RenderError(w, r, 404, "Project not found")
		return
	}

	if err = s.projects.Delete(id); err != nil {
		s.render.RenderError(w, r, 500, "Failed to delete project")
		return
	}

	s.deployer.RemoveProjectDir(p)

	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}
