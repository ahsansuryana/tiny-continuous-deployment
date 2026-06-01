package main

import (
	"deployhub/internal/auth"
	"deployhub/internal/database"
	"deployhub/internal/deploy"
	"deployhub/internal/docker"
	srvhttp "deployhub/internal/http"
	"deployhub/internal/project"
	"deployhub/internal/settings"
	"embed"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/sqlite3store"
)

//go:embed web/templates
//go:embed web/static
var assets embed.FS

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	adminUser := os.Getenv("ADMIN_USERNAME")
	adminPass := os.Getenv("ADMIN_PASSWORD")
	if adminUser == "" || adminPass == "" {
		log.Fatal("ADMIN_USERNAME and ADMIN_PASSWORD environment variables are required")
	}

	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "/data"
	}

	db, err := database.New(dataDir + "/deployhub.db" +
		"?_pragma=journal_mode(WAL)" +
		"&_pragma=busy_timeout(5000)" +
		"&_pragma=synchronous(NORMAL)" +
		"&_pragma=foreign_keys(ON)" +
		"&_pragma=cache_size(-8000)")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	sm := scs.New()
	sm.Store = sqlite3store.NewWithCleanupInterval(db.DB, 1*time.Hour)
	sm.Lifetime = 24 * time.Hour
	sm.Cookie.Persist = true
	sm.Cookie.Secure = false
	sm.Cookie.HttpOnly = true
	sm.Cookie.SameSite = http.SameSiteLaxMode

	au, err := auth.New(sm, adminUser, adminPass)
	if err != nil {
		log.Fatalf("failed to initialize auth: %v", err)
	}

	projectSvc := project.New(db)
	settingsSvc := settings.New(db)

	settingsCfg, err := settingsSvc.Get()
	if err != nil {
		log.Fatalf("failed to load settings: %v", err)
	}

	dockerClient := docker.New(settingsCfg.DockerSocket)

	deployer := deploy.New(projectSvc, dockerClient, db, settingsSvc)

	srv := srvhttp.New(au, projectSvc, deployer, dockerClient, settingsSvc, assets)

	handler := srv.Handler()
	handler = sm.LoadAndSave(handler)

	log.Printf("starting deployhub on %s", addr)
	log.Printf("data directory: %s", dataDir)

	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
