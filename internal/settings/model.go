package settings

type Settings struct {
	AppURL          string `json:"app_url"`
	Timezone        string `json:"timezone"`
	DockerSocket    string `json:"docker_socket"`
	DeployCommand   string `json:"deploy_command"`
	DeployTimeout   int    `json:"deploy_timeout"`
	WebhookSecret   string `json:"webhook_secret"`
	TraefikHostname string `json:"traefik_hostname"`
}

func Defaults() *Settings {
	return &Settings{
		AppURL:          "http://localhost:8080",
		Timezone:        "UTC",
		DockerSocket:    "/var/run/docker.sock",
		DeployCommand:   "docker compose pull && docker compose up -d",
		DeployTimeout:   300,
		TraefikHostname: "deploy.example.com",
	}
}
