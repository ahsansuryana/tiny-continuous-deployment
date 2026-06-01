package http

import (
	"deployhub/internal/deploy"
	"net/http"
	"strconv"
)

type logViewerData struct {
	ProjectName string
	ProjectID   int64
	Deployment  *deploy.Deployment
	History     []deploy.Deployment
	CSRFToken   string
	Username    string
}

func (s *Server) projectLogs(w http.ResponseWriter, r *http.Request) {
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

	history, err := s.deployer.GetDeploymentsByProject(id, 10)
	if err != nil {
		history = nil
	}

	var current *deploy.Deployment
	if len(history) > 0 {
		current = &history[0]
	}

	username := s.auth.GetUsername(r)
	s.render.Render(w, r, "log_viewer.html", logViewerData{
		ProjectName: p.Name,
		ProjectID:   id,
		Deployment:  current,
		History:     history,
		CSRFToken:   s.csrfToken(r),
		Username:    username,
	})
}

func (s *Server) projectLogStream(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	deployParam := r.URL.Query().Get("deploy")
	currentID := int64(0)
	if deployParam != "" {
		currentID, _ = strconv.ParseInt(deployParam, 10, 64)
	}

	if currentID == 0 {
		history, err := s.deployer.GetDeploymentsByProject(id, 1)
		if err != nil || len(history) == 0 {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<div class="text-zinc-400 p-4 text-sm">No deployments yet</div>`))
			return
		}
		currentID = history[0].ID
	}

	dep, err := s.deployer.GetDeployment(currentID)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<div class="text-zinc-400 p-4 text-sm">Deployment not found</div>`))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	statusClass := "text-zinc-400"
	switch dep.Status {
	case "success":
		statusClass = "text-green-500"
	case "failed":
		statusClass = "text-red-500"
	case "running":
		statusClass = "text-amber-500"
	}

	_, _ = w.Write([]byte(`<div class="font-mono text-sm leading-relaxed whitespace-pre-wrap">`))

	if dep.Log != "" {
		lines := splitLines(dep.Log)
		for _, line := range lines {
			_, _ = w.Write([]byte(`<div class="hover:bg-zinc-800/50 px-4 py-0.5">`))
			_, _ = w.Write([]byte(escapeHTML(line)))
			_, _ = w.Write([]byte(`</div>`))
		}
	}

	_, _ = w.Write([]byte(`</div>`))

	_, _ = w.Write([]byte(`<div class="mt-4 px-4 py-3 border-t border-zinc-800 flex items-center gap-2 text-sm">`))
	_, _ = w.Write([]byte(`<span class="inline-block w-2 h-2 rounded-full `))
	if dep.Status == "running" {
		_, _ = w.Write([]byte(`bg-amber-500`))
	} else if dep.Status == "success" {
		_, _ = w.Write([]byte(`bg-green-500`))
	} else if dep.Status == "failed" {
		_, _ = w.Write([]byte(`bg-red-500`))
	} else {
		_, _ = w.Write([]byte(`bg-zinc-500`))
	}
	_, _ = w.Write([]byte(`"></span>`))
	_, _ = w.Write([]byte(`<span class="` + statusClass + ` font-medium">` + dep.Status + `</span>`))
	if dep.FinishedAt != "" {
		_, _ = w.Write([]byte(`<span class="text-zinc-400">· ` + dep.FinishedAt + `</span>`))
	}
	_, _ = w.Write([]byte(`</div>`))

	if dep.Status == "running" {
		w.Header().Set("HX-Trigger", `{"pollLogs":true}`)
	}
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	current := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, string(current))
			current = current[:0]
		} else {
			current = append(current, s[i])
		}
	}
	if len(current) > 0 {
		lines = append(lines, string(current))
	}
	return lines
}

func escapeHTML(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '&':
			b = append(b, "&amp;"...)
		case '<':
			b = append(b, "&lt;"...)
		case '>':
			b = append(b, "&gt;"...)
		case '"':
			b = append(b, "&quot;"...)
		case '\'':
			b = append(b, "&#39;"...)
		default:
			b = append(b, s[i])
		}
	}
	return string(b)
}
