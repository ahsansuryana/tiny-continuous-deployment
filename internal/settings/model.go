package settings

type Settings struct {
	AppURL              string `json:"app_url"`
	Timezone            string `json:"timezone"`
	DockerSocket        string `json:"docker_socket"`
	DeployCommand       string `json:"deploy_command"`
	DeployTimeout       int    `json:"deploy_timeout"`
	WebhookSecret       string `json:"webhook_secret"`
	TraefikDomain       string `json:"traefik_domain"`
	TraefikEmail        string `json:"traefik_email"`
	TraefikHTTPEntrypoint  string `json:"traefik_http_entrypoint"`
	TraefikHTTPSEntrypoint string `json:"traefik_https_entrypoint"`
	TraefikNetwork      string `json:"traefik_network"`
}

func Defaults() *Settings {
	return &Settings{
		AppURL:              "http://localhost:8080",
		Timezone:            "UTC",
		DockerSocket:        "/var/run/docker.sock",
		DeployCommand:       "docker compose pull && docker compose up -d",
		DeployTimeout:       300,
		TraefikDomain:       "mydomain.com",
		TraefikEmail:        "admin@mydomain.com",
		TraefikHTTPEntrypoint:  "web",
		TraefikHTTPSEntrypoint: "websecure",
		TraefikNetwork:      "traefik",
	}
}
