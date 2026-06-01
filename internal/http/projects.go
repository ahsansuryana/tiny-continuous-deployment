package http

import (
	"deployhub/internal/project"
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
	Project    *project.Project
	Status     string
	LastDeploy string
	DeployID   int64
	CSRFToken  string
	Username   string
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

	p := &project.Project{
		Name:         strings.TrimSpace(r.FormValue("name")),
		RepoURL:      strings.TrimSpace(r.FormValue("repo_url")),
		ComposePath:  strings.TrimSpace(r.FormValue("compose_path")),
		WebhookToken: strings.TrimSpace(r.FormValue("webhook_token")),
		Branch:       strings.TrimSpace(r.FormValue("branch")),
		RegistryType: strings.TrimSpace(r.FormValue("registry_type")),
		RegistryUser: strings.TrimSpace(r.FormValue("registry_username")),
		RegistryPass: strings.TrimSpace(r.FormValue("registry_password")),
		AutoDeploy:   r.FormValue("auto_deploy") == "on",
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
	s.render.Render(w, r, "project_detail.html", projectDetailData{
		Project:    p,
		Status:     status,
		LastDeploy: lastDeploy,
		DeployID:   deployID,
		CSRFToken:  s.csrfToken(r),
		Username:   username,
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

	p := &project.Project{
		ID:           id,
		Name:         strings.TrimSpace(r.FormValue("name")),
		RepoURL:      strings.TrimSpace(r.FormValue("repo_url")),
		ComposePath:  strings.TrimSpace(r.FormValue("compose_path")),
		WebhookToken: strings.TrimSpace(r.FormValue("webhook_token")),
		Branch:       strings.TrimSpace(r.FormValue("branch")),
		RegistryType: strings.TrimSpace(r.FormValue("registry_type")),
		RegistryUser: strings.TrimSpace(r.FormValue("registry_username")),
		RegistryPass: strings.TrimSpace(r.FormValue("registry_password")),
		AutoDeploy:   r.FormValue("auto_deploy") == "on",
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

func (s *Server) projectDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.render.RenderError(w, r, 400, "Invalid project ID")
		return
	}

	if err := s.projects.Delete(id); err != nil {
		s.render.RenderError(w, r, 500, "Failed to delete project")
		return
	}

	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}
