// Command phishforge is the single binary for the PhishForge platform. It runs in
// one of three modes selected by PHISHFORGE_MODE or the first CLI arg:
//
//	migrate  apply database migrations and exit
//	api      serve the admin API + SPA and the phishing/tracking server
//	worker   consume launch jobs and send campaign email
//
// PhishForge is an advanced phishing simulation & security awareness platform.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/furkan-enes-polatoglu/phishforge/internal/api"
	"github.com/furkan-enes-polatoglu/phishforge/internal/auth"
	"github.com/furkan-enes-polatoglu/phishforge/internal/config"
	"github.com/furkan-enes-polatoglu/phishforge/internal/migrate"
	"github.com/furkan-enes-polatoglu/phishforge/internal/models"
	"github.com/furkan-enes-polatoglu/phishforge/internal/phishing"
	"github.com/furkan-enes-polatoglu/phishforge/internal/queue"
	"github.com/furkan-enes-polatoglu/phishforge/internal/store"
	"github.com/furkan-enes-polatoglu/phishforge/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	// Allow "phishforge <mode>" to override PHISHFORGE_MODE.
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		cfg.Mode = os.Args[1]
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	switch cfg.Mode {
	case "migrate":
		runMigrate(ctx, cfg)
	case "worker":
		runWorker(ctx, cfg)
	case "api", "serve":
		runAPI(ctx, cfg)
	default:
		log.Fatalf("unknown mode %q (want migrate|api|worker)", cfg.Mode)
	}
}

func openStore(ctx context.Context, cfg *config.Config) *store.Store {
	// Retry briefly so containers can start before Postgres is ready.
	var st *store.Store
	var err error
	for i := 0; i < 30; i++ {
		st, err = store.New(ctx, cfg.DatabaseURL)
		if err == nil {
			return st
		}
		log.Printf("waiting for database (%d/30): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	log.Fatalf("database unreachable: %v", err)
	return nil
}

func openQueue(ctx context.Context, cfg *config.Config) *queue.Queue {
	q, err := queue.New(cfg.RedisURL)
	if err != nil {
		log.Fatalf("queue: %v", err)
	}
	for i := 0; i < 30; i++ {
		if err = q.Ping(ctx); err == nil {
			return q
		}
		log.Printf("waiting for redis (%d/30): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	log.Fatalf("redis unreachable: %v", err)
	return nil
}

func runMigrate(ctx context.Context, cfg *config.Config) {
	st := openStore(ctx, cfg)
	defer st.Close()
	if err := migrate.Run(ctx, st.Pool()); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Println("migrations up to date")
}

func runWorker(ctx context.Context, cfg *config.Config) {
	st := openStore(ctx, cfg)
	defer st.Close()
	q := openQueue(ctx, cfg)
	defer q.Close()
	w := worker.New(cfg, st, q)
	if err := w.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("worker: %v", err)
	}
}

func runAPI(ctx context.Context, cfg *config.Config) {
	st := openStore(ctx, cfg)
	defer st.Close()
	// Ensure schema exists (idempotent) so a single `api` container is enough in dev.
	if err := migrate.Run(ctx, st.Pool()); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	bootstrapAdmin(ctx, cfg, st)

	q := openQueue(ctx, cfg)
	defer q.Close()

	adminSrv := &http.Server{Addr: cfg.AdminAddr, Handler: api.NewServer(cfg, st, q).Router()}
	phishSrv := &http.Server{Addr: cfg.PhishAddr, Handler: phishing.New(cfg, st).Router()}

	go func() {
		log.Printf("admin API listening on %s", cfg.AdminAddr)
		if err := adminSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("admin server: %v", err)
		}
	}()
	go func() {
		log.Printf("phishing/tracking server listening on %s", cfg.PhishAddr)
		if err := phishSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("phishing server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")
	shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = adminSrv.Shutdown(shCtx)
	_ = phishSrv.Shutdown(shCtx)
}

// bootstrapAdmin creates the first org + admin user when the DB has no users and
// bootstrap credentials are configured. Idempotent and safe to run every start.
func bootstrapAdmin(ctx context.Context, cfg *config.Config, st *store.Store) {
	n, err := st.CountUsers(ctx)
	if err != nil {
		log.Printf("bootstrap: count users: %v", err)
		return
	}
	if n > 0 {
		return
	}
	if cfg.BootstrapAdminUsername == "" || cfg.BootstrapAdminPass == "" {
		log.Println("bootstrap: no users yet; set BOOTSTRAP_ADMIN_USERNAME and BOOTSTRAP_ADMIN_PASSWORD to create the first admin")
		return
	}
	org, err := st.CreateOrg(ctx, cfg.BootstrapOrgName)
	if err != nil {
		log.Printf("bootstrap: create org: %v", err)
		return
	}
	hash, err := auth.HashPassword(cfg.BootstrapAdminPass)
	if err != nil {
		log.Printf("bootstrap: hash: %v", err)
		return
	}
	u, err := st.CreateUser(ctx, org.ID, strings.ToLower(cfg.BootstrapAdminUsername), hash, models.RoleAdmin)
	if err != nil {
		log.Printf("bootstrap: create user: %v", err)
		return
	}
	log.Printf("bootstrap: created org %q and admin %s", org.Name, u.Username)
}
