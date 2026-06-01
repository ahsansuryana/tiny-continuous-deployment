package http

import (
	"encoding/json"
	"net/http"
	"strings"
)

type webhookResponse struct {
	Success      bool   `json:"success"`
	DeploymentID int64  `json:"deployment_id,omitempty"`
	Error        string `json:"error,omitempty"`
}

func (s *Server) webhookDeploy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(webhookResponse{Success: false, Error: "missing authorization header"})
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(webhookResponse{Success: false, Error: "invalid authorization format"})
		return
	}

	projectName := r.PathValue("project")
	if projectName == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(webhookResponse{Success: false, Error: "missing project name"})
		return
	}

	p, err := s.projects.GetByName(projectName)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(webhookResponse{Success: false, Error: "project not found"})
		return
	}

	if p.WebhookToken != token {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(webhookResponse{Success: false, Error: "invalid token"})
		return
	}

	if s.deployer.IsRunning(p.ID) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(webhookResponse{Success: false, Error: "deployment already in progress"})
		return
	}

	deployID, err := s.deployer.Run(p.ID, "webhook")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(webhookResponse{Success: false, Error: err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(webhookResponse{
		Success:      true,
		DeploymentID: deployID,
	})
}
