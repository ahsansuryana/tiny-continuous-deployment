package project

type Project struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	RepoURL        string `json:"repo_url"`
	ComposePath    string `json:"compose_path"`
	WebhookToken   string `json:"-"`
	Branch         string `json:"branch"`
	RegistryType   string `json:"registry_type"`
	RegistryUser   string `json:"registry_username"`
	RegistryPass   string `json:"-"`
	AutoDeploy     bool   `json:"auto_deploy"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type ProjectStatus struct {
	Project
	Status        string `json:"status"`
	LastDeployAt  string `json:"last_deploy_at"`
	DeploymentID  int64  `json:"deployment_id"`
}

func (p *Project) Validate() string {
	if p.Name == "" {
		return "name is required"
	}
	if len(p.Name) > 128 {
		return "name must be 128 characters or less"
	}
	if p.ComposePath == "" {
		return "compose path is required"
	}
	if p.WebhookToken == "" {
		return "webhook token is required"
	}
	return ""
}
