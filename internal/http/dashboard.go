package http

import (
	"net/http"
)

type dashboardData struct {
	TotalProjects int
	RunningCount  int
	FailedCount   int
	LastDeploy    string
	Projects      []projectStatusView
	CSRFToken     string
	Username      string
}

type projectStatusView struct {
	ID           int64
	Name         string
	Status       string
	ComposePath  string
	LastDeployAt string
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	projects, err := s.projects.ListWithStatus()
	if err != nil {
		s.render.RenderError(w, r, 500, "Failed to load projects")
		return
	}

	var total, running, failed int
	var lastDeploy string
	var views []projectStatusView

	for _, p := range projects {
		total++
		switch p.Status {
		case "running":
			running++
		case "failed":
			failed++
		case "success":
		}
		if p.Status != "" && (lastDeploy == "" || p.LastDeployAt > lastDeploy) {
			lastDeploy = p.LastDeployAt
		}
		views = append(views, projectStatusView{
			ID:           p.ID,
			Name:         p.Name,
			Status:       p.Status,
			ComposePath:  p.ComposePath,
			LastDeployAt: p.LastDeployAt,
		})
	}

	username := s.auth.GetUsername(r)

	s.render.Render(w, r, "dashboard.html", dashboardData{
		TotalProjects: total,
		RunningCount:  running,
		FailedCount:   failed,
		LastDeploy:    lastDeploy,
		Projects:      views,
		CSRFToken:     s.csrfToken(r),
		Username:      username,
	})
}
