package settings

import (
	"deployhub/internal/database"
	"strconv"
)

type Service struct {
	db *database.DB
}

func New(db *database.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Get() (*Settings, error) {
	settings := Defaults()

	rows, err := s.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		switch key {
		case "app_url":
			settings.AppURL = value
		case "timezone":
			settings.Timezone = value
		case "docker_socket":
			settings.DockerSocket = value
		case "deploy_command":
			settings.DeployCommand = value
		case "deploy_timeout":
			if t, err := strconv.Atoi(value); err == nil {
				settings.DeployTimeout = t
			}
		case "webhook_secret":
			settings.WebhookSecret = value
		case "traefik_hostname", "traefik_domain":
			settings.TraefikDomain = value
		case "traefik_email":
			settings.TraefikEmail = value
		case "traefik_http_entrypoint":
			settings.TraefikHTTPEntrypoint = value
		case "traefik_https_entrypoint":
			settings.TraefikHTTPSEntrypoint = value
		case "traefik_network":
			settings.TraefikNetwork = value
		}
	}
	return settings, nil
}

func (s *Service) Update(settings *Settings) error {
	pairs := map[string]string{
		"app_url":                settings.AppURL,
		"timezone":               settings.Timezone,
		"docker_socket":          settings.DockerSocket,
		"deploy_command":         settings.DeployCommand,
		"deploy_timeout":         strconv.Itoa(settings.DeployTimeout),
		"webhook_secret":         settings.WebhookSecret,
		"traefik_domain":         settings.TraefikDomain,
		"traefik_email":          settings.TraefikEmail,
		"traefik_http_entrypoint":  settings.TraefikHTTPEntrypoint,
		"traefik_https_entrypoint": settings.TraefikHTTPSEntrypoint,
		"traefik_network":        settings.TraefikNetwork,
	}

	for key, value := range pairs {
		_, err := s.db.Exec(
			`INSERT INTO settings (key, value) VALUES (?, ?)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
			key, value)
		if err != nil {
			return err
		}
	}
	return nil
}
