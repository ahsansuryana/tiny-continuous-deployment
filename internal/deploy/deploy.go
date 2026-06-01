package deploy

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"deployhub/internal/database"
	"deployhub/internal/docker"
	"deployhub/internal/project"
	"deployhub/internal/settings"
)

type Deployer struct {
	projects *project.Service
	docker   *docker.Client
	db       *database.DB
	settings *settings.Service

	mu       sync.Mutex
	running  map[int64]context.CancelFunc
}

func New(ps *project.Service, dc *docker.Client, db *database.DB, ss *settings.Service) *Deployer {
	return &Deployer{
		projects: ps,
		docker:   dc,
		db:       db,
		settings: ss,
		running:  make(map[int64]context.CancelFunc),
	}
}

type Deployment struct {
	ID         int64  `json:"id"`
	ProjectID  int64  `json:"project_id"`
	Status     string `json:"status"`
	Trigger    string `json:"trigger"`
	StartedAt  string `json:"started_at"`
	FinishedAt string `json:"finished_at"`
	Log        string `json:"log"`
}

func (d *Deployer) IsRunning(projectID int64) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, ok := d.running[projectID]
	return ok
}

func (d *Deployer) Run(projectID int64, trigger string) (int64, error) {
	d.mu.Lock()
	if _, ok := d.running[projectID]; ok {
		d.mu.Unlock()
		return 0, fmt.Errorf("deployment already in progress for this project")
	}
	ctx, cancel := context.WithCancel(context.Background())
	d.running[projectID] = cancel
	d.mu.Unlock()

	proj, err := d.projects.GetByID(projectID)
	if err != nil {
		cancel()
		d.clearRunning(projectID)
		return 0, fmt.Errorf("project not found: %w", err)
	}

	settings, err := d.settings.Get()
	if err != nil {
		cancel()
		d.clearRunning(projectID)
		return 0, fmt.Errorf("failed to get settings: %w", err)
	}

	result, err := d.db.Exec(
		`INSERT INTO deployments (project_id, status, trigger) VALUES (?, 'running', ?)`,
		projectID, trigger)
	if err != nil {
		cancel()
		d.clearRunning(projectID)
		return 0, err
	}

	deployID, _ := result.LastInsertId()

	go func() {
		defer d.clearRunning(projectID)
		if err := d.ensureComposeFile(proj); err != nil {
			d.finishDeployment(deployID, "failed", fmt.Sprintf("[ERROR] %v", err))
			return
		}
		d.injectTraefikLabels(proj, settings)
		d.executeDeploy(ctx, deployID, proj, settings)
	}()

	return deployID, nil
}

func (d *Deployer) Cancel(projectID int64) {
	d.mu.Lock()
	cancel, ok := d.running[projectID]
	d.mu.Unlock()
	if ok {
		cancel()
	}
}

func (d *Deployer) executeDeploy(ctx context.Context, deployID int64, proj *project.Project, s *settings.Settings) {
	var timeout time.Duration
	if s.DeployTimeout > 0 {
		timeout = time.Duration(s.DeployTimeout) * time.Second
	} else {
		timeout = 300 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	logBuilder := NewLogBuilder()

	logBuilder.Append("[INFO] Starting deployment for %s (%s)", proj.Name, proj.DeployDir())
	d.updateDeploymentLog(deployID, logBuilder.String())

	if s.DeployCommand == "" {
		s.DeployCommand = "docker compose pull && docker compose up -d"
	}

	logBuilder.Append("[INFO] Running: %s", s.DeployCommand)
	d.updateDeploymentLog(deployID, logBuilder.String())

	err := d.docker.RunCommand(ctx, proj.DeployDir(), s.DeployCommand, func(line string) {
		logBuilder.Append("%s", line)
	})

	if err != nil {
		logBuilder.Append("[ERROR] %v", err)
		d.updateDeploymentLog(deployID, logBuilder.String())
		d.finishDeployment(deployID, "failed", logBuilder.String())
		return
	}

	logBuilder.Append("[SUCCESS] Deployment completed successfully")
	d.updateDeploymentLog(deployID, logBuilder.String())
	d.finishDeployment(deployID, "success", logBuilder.String())
}

func (d *Deployer) updateDeploymentLog(deployID int64, logText string) {
	_, err := d.db.Exec(`UPDATE deployments SET log = ? WHERE id = ?`, logText, deployID)
	if err != nil {
		log.Printf("failed to update deployment log: %v", err)
	}
}

func (d *Deployer) finishDeployment(deployID int64, status string, logText string) {
	_, err := d.db.Exec(
		`UPDATE deployments SET status = ?, finished_at = datetime('now'), log = ? WHERE id = ?`,
		status, logText, deployID)
	if err != nil {
		log.Printf("failed to finish deployment: %v", err)
	}
}

func (d *Deployer) injectTraefikLabels(proj *project.Project, s *settings.Settings) {
	if proj.TraefikHostname == "" || s.TraefikDomain == "" {
		return
	}

	composePath := filepath.Join(proj.DeployDir(), "docker-compose.yml")
	data, err := os.ReadFile(composePath)
	if err != nil {
		log.Printf("failed to read compose file for label injection: %v", err)
		return
	}

	content := string(data)
	domain := s.TraefikDomain
	port := proj.TraefikPort
	if port == 0 {
		port = 80
	}
	entrypoint := s.TraefikHTTPSEntrypoint
	if entrypoint == "" {
		entrypoint = "websecure"
	}
	network := s.TraefikNetwork
	if network == "" {
		network = "traefik"
	}

	serviceName := strings.ReplaceAll(proj.Name, "_", "-")

	labels := fmt.Sprintf(`
    labels:
      - traefik.enable=true
      - traefik.http.routers.%s.rule=Host(\`+"`"+`%s.%s\`+"`"+`)
      - traefik.http.routers.%s.entrypoints=%s
      - traefik.http.routers.%s.tls=true
      - traefik.http.routers.%s.tls.certresolver=letsencrypt
      - traefik.http.services.%s.loadbalancer.server.port=%d`, serviceName, proj.TraefikHostname, domain, serviceName, entrypoint, serviceName, serviceName, serviceName, port)

	middleware := strings.TrimSpace(proj.TraefikMiddleware)
	if middleware != "" {
		labels += fmt.Sprintf(`
      - traefik.http.routers.%s.middlewares=%s`, serviceName, middleware)
	}

	networkConfig := fmt.Sprintf(`
    networks:
      - %s`, network)

	topNetworks := fmt.Sprintf(`
networks:
  %s:
    external: true`, network)

	modified := injectIntoCompose(content, labels, networkConfig, topNetworks)

	if err := os.WriteFile(composePath, []byte(modified), 0644); err != nil {
		log.Printf("failed to write compose file with labels: %v", err)
	}
}

func injectIntoCompose(content, labels, networkConfig, topNetworks string) string {
	lines := strings.Split(content, "\n")
	var result []string
	servicesStarted := false
	hasNetworksSection := false
	serviceIndices := []int{}
	lastServiceLine := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "services:" {
			servicesStarted = true
		}
		if servicesStarted && len(trimmed) > 0 && trimmed[0] != '#' && trimmed[0] != ' ' && trimmed != "services:" {
			if i > 0 && strings.TrimSpace(lines[i-1]) != "" {
				serviceIndices = append(serviceIndices, i)
			}
		}
		if strings.HasPrefix(trimmed, "networks:") {
			hasNetworksSection = true
		}
		result = append(result, line)
	}

	if len(serviceIndices) == 0 {
		return content
	}

	lastServiceLine = serviceIndices[len(serviceIndices)-1]

	for i := lastServiceLine; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if trimmed[0] != ' ' {
			lastServiceLine = i
			break
		}
	}

	var finalResult []string
	for i, line := range lines {
		finalResult = append(finalResult, line)
		if i == lastServiceLine-1 {
			finalResult = append(finalResult, labels, networkConfig)
		}
	}

	if !hasNetworksSection {
		finalResult = append(finalResult, topNetworks)
	}

	return strings.Join(finalResult, "\n")
}

func (d *Deployer) ensureComposeFile(proj *project.Project) error {
	if proj.ComposeType != "inline" {
		return nil
	}
	dir := "/data/projects/" + proj.Name
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(proj.ComposeContent), 0644); err != nil {
		return fmt.Errorf("failed to write docker-compose.yml: %w", err)
	}
	return nil
}

func (d *Deployer) RemoveProjectDir(proj *project.Project) {
	if proj.ComposeType != "inline" {
		return
	}
	os.RemoveAll("/data/projects/" + proj.Name)
}

func (d *Deployer) clearRunning(projectID int64) {
	d.mu.Lock()
	delete(d.running, projectID)
	d.mu.Unlock()
}

func (d *Deployer) GetDeployment(deployID int64) (*Deployment, error) {
	dep := &Deployment{}
	err := d.db.QueryRow(
		`SELECT id, project_id, status, trigger, started_at, COALESCE(finished_at,''), log
		 FROM deployments WHERE id = ?`, deployID).Scan(
		&dep.ID, &dep.ProjectID, &dep.Status, &dep.Trigger,
		&dep.StartedAt, &dep.FinishedAt, &dep.Log)
	if err != nil {
		return nil, err
	}
	return dep, nil
}

func (d *Deployer) GetDeploymentsByProject(projectID int64, limit int) ([]Deployment, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := d.db.Query(
		`SELECT id, project_id, status, trigger, started_at, COALESCE(finished_at,''), log
		 FROM deployments WHERE project_id = ?
		 ORDER BY started_at DESC LIMIT ?`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deploys []Deployment
	for rows.Next() {
		var dep Deployment
		if err := rows.Scan(&dep.ID, &dep.ProjectID, &dep.Status, &dep.Trigger,
			&dep.StartedAt, &dep.FinishedAt, &dep.Log); err != nil {
			return nil, err
		}
		deploys = append(deploys, dep)
	}
	return deploys, nil
}

type LogBuilder struct {
	log string
	mu  sync.Mutex
}

func NewLogBuilder() *LogBuilder {
	return &LogBuilder{}
}

func (lb *LogBuilder) Append(format string, args ...interface{}) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	line := fmt.Sprintf(format, args...)
	timestamp := time.Now().UTC().Format("15:04:05")
	if lb.log != "" {
		lb.log += "\n"
	}
	lb.log += fmt.Sprintf("[%s] %s", timestamp, line)
}

func (lb *LogBuilder) String() string {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	return lb.log
}
