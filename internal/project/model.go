package project

type Project struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	RepoURL        string `json:"repo_url"`
	ComposePath    string `json:"compose_path"`
	ComposeType    string `json:"compose_type"`
	ComposeContent string `json:"compose_content"`
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
	Status       string `json:"status"`
	LastDeployAt string `json:"last_deploy_at"`
	DeploymentID int64  `json:"deployment_id"`
}

func (p *Project) Validate() string {
	if p.Name == "" {
		return "name is required"
	}
	if len(p.Name) > 128 {
		return "name must be 128 characters or less"
	}
	if p.ComposeType != "inline" && p.ComposeType != "path" {
		p.ComposeType = "path"
	}
	if p.ComposeType == "path" && p.ComposePath == "" {
		return "compose path is required when using file path"
	}
	if p.ComposeType == "inline" && p.ComposeContent == "" {
		return "compose content is required when using inline YAML"
	}
	return ""
}

func (p *Project) DeployDir() string {
	if p.ComposeType == "inline" {
		return "/data/projects/" + p.Name
	}
	return p.ComposePath
}
