package http

import (
	"deployhub/internal/deploy"
	"net/http"
	"strconv"
)

func (s *Server) projectDeploy(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.render.RenderError(w, r, 400, "Invalid project ID")
		return
	}

	if s.deployer.IsRunning(id) {
		http.Redirect(w, r, "/projects/"+strconv.FormatInt(id, 10)+"?error=deploy+running", http.StatusSeeOther)
		return
	}

	deployID, err := s.deployer.Run(id, "manual")
	if err != nil {
		s.render.RenderError(w, r, 500, "Deploy failed: "+err.Error())
		return
	}

	_ = deployID
	http.Redirect(w, r, "/projects/"+strconv.FormatInt(id, 10)+"/logs", http.StatusSeeOther)
}
