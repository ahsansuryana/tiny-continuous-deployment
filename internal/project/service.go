package project

import (
	"crypto/rand"
	"encoding/base64"
	"deployhub/internal/database"
)

type Service struct {
	db *database.DB
}

func New(db *database.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List() ([]Project, error) {
	rows, err := s.db.Query(
		`SELECT id, name, repo_url, compose_path, compose_type, compose_content,
		        webhook_token, branch, registry_type, registry_user, registry_pass,
		        auto_deploy, traefik_hostname, traefik_port, traefik_middleware,
		        created_at, updated_at
		 FROM projects ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.RepoURL, &p.ComposePath,
			&p.ComposeType, &p.ComposeContent, &p.WebhookToken, &p.Branch,
			&p.RegistryType, &p.RegistryUser, &p.RegistryPass,
			&p.AutoDeploy, &p.TraefikHostname, &p.TraefikPort, &p.TraefikMiddleware,
			&p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func (s *Service) ListWithStatus() ([]ProjectStatus, error) {
	rows, err := s.db.Query(
		`SELECT p.id, p.name, p.repo_url, p.compose_path, p.compose_type, p.compose_content,
		        p.webhook_token, p.branch, p.registry_type, p.registry_user, p.registry_pass,
		        p.auto_deploy, p.traefik_hostname, p.traefik_port, p.traefik_middleware,
		        p.created_at, p.updated_at,
		        COALESCE(d.status, ''), COALESCE(d.started_at, ''), COALESCE(d.id, 0)
		 FROM projects p
		 LEFT JOIN deployments d ON d.id = (
		     SELECT id FROM deployments WHERE project_id = p.id ORDER BY started_at DESC LIMIT 1
		 )
		 ORDER BY p.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ProjectStatus
	for rows.Next() {
		var ps ProjectStatus
		if err := rows.Scan(&ps.ID, &ps.Name, &ps.RepoURL, &ps.ComposePath,
			&ps.ComposeType, &ps.ComposeContent, &ps.WebhookToken, &ps.Branch,
			&ps.RegistryType, &ps.RegistryUser, &ps.RegistryPass,
			&ps.AutoDeploy, &ps.TraefikHostname, &ps.TraefikPort, &ps.TraefikMiddleware,
			&ps.CreatedAt, &ps.UpdatedAt,
			&ps.Status, &ps.LastDeployAt, &ps.DeploymentID); err != nil {
			return nil, err
		}
		results = append(results, ps)
	}
	return results, nil
}

func (s *Service) GetByID(id int64) (*Project, error) {
	p := &Project{}
	err := s.db.QueryRow(
		`SELECT id, name, repo_url, compose_path, compose_type, compose_content,
		        webhook_token, branch, registry_type, registry_user, registry_pass,
		        auto_deploy, traefik_hostname, traefik_port, traefik_middleware,
		        created_at, updated_at
		 FROM projects WHERE id = ?`, id).Scan(
		&p.ID, &p.Name, &p.RepoURL, &p.ComposePath,
		&p.ComposeType, &p.ComposeContent, &p.WebhookToken, &p.Branch,
		&p.RegistryType, &p.RegistryUser, &p.RegistryPass,
		&p.AutoDeploy, &p.TraefikHostname, &p.TraefikPort, &p.TraefikMiddleware,
		&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) GetByToken(token string) (*Project, error) {
	p := &Project{}
	err := s.db.QueryRow(
		`SELECT id, name, repo_url, compose_path, compose_type, compose_content,
		        webhook_token, branch, registry_type, registry_user, registry_pass,
		        auto_deploy, traefik_hostname, traefik_port, traefik_middleware,
		        created_at, updated_at
		 FROM projects WHERE webhook_token = ?`, token).Scan(
		&p.ID, &p.Name, &p.RepoURL, &p.ComposePath,
		&p.ComposeType, &p.ComposeContent, &p.WebhookToken, &p.Branch,
		&p.RegistryType, &p.RegistryUser, &p.RegistryPass,
		&p.AutoDeploy, &p.TraefikHostname, &p.TraefikPort, &p.TraefikMiddleware,
		&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) GetByName(name string) (*Project, error) {
	p := &Project{}
	err := s.db.QueryRow(
		`SELECT id, name, repo_url, compose_path, compose_type, compose_content,
		        webhook_token, branch, registry_type, registry_user, registry_pass,
		        auto_deploy, traefik_hostname, traefik_port, traefik_middleware,
		        created_at, updated_at
		 FROM projects WHERE name = ?`, name).Scan(
		&p.ID, &p.Name, &p.RepoURL, &p.ComposePath,
		&p.ComposeType, &p.ComposeContent, &p.WebhookToken, &p.Branch,
		&p.RegistryType, &p.RegistryUser, &p.RegistryPass,
		&p.AutoDeploy, &p.TraefikHostname, &p.TraefikPort, &p.TraefikMiddleware,
		&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) Create(p *Project) error {
	if p.WebhookToken == "" {
		p.WebhookToken = generateToken()
	}
	_, err := s.db.Exec(
		`INSERT INTO projects (name, repo_url, compose_path, compose_type, compose_content,
		                       webhook_token, branch, registry_type, registry_user,
		                       registry_pass, auto_deploy,
		                       traefik_hostname, traefik_port, traefik_middleware)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Name, p.RepoURL, p.ComposePath, p.ComposeType, p.ComposeContent,
		p.WebhookToken, p.Branch, p.RegistryType, p.RegistryUser,
		p.RegistryPass, boolToInt(p.AutoDeploy),
		p.TraefikHostname, p.TraefikPort, p.TraefikMiddleware)
	return err
}

func (s *Service) Update(p *Project) error {
	_, err := s.db.Exec(
		`UPDATE projects SET name=?, repo_url=?, compose_path=?, compose_type=?,
		                     compose_content=?, webhook_token=?, branch=?,
		                     registry_type=?, registry_user=?, registry_pass=?,
		                     auto_deploy=?, traefik_hostname=?, traefik_port=?,
		                     traefik_middleware=?, updated_at=datetime('now')
		 WHERE id=?`,
		p.Name, p.RepoURL, p.ComposePath, p.ComposeType, p.ComposeContent,
		p.WebhookToken, p.Branch, p.RegistryType, p.RegistryUser,
		p.RegistryPass, boolToInt(p.AutoDeploy),
		p.TraefikHostname, p.TraefikPort, p.TraefikMiddleware, p.ID)
	return err
}

func (s *Service) Delete(id int64) error {
	_, err := s.db.Exec(`DELETE FROM deployments WHERE project_id = ?`, id)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM projects WHERE id = ?`, id)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
