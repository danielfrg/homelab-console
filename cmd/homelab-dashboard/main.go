package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"homelab-dashboard/internal/apps"
	"homelab-dashboard/internal/httpapi"
	"homelab-dashboard/internal/web"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	assets, err := web.Assets()
	if err != nil {
		log.Fatal(err)
	}
	config, err := apps.KubernetesConfig()
	if err != nil {
		log.Fatal(err)
	}
	source, err := apps.NewKubernetesSource(config)
	if err != nil {
		log.Fatal(err)
	}
	catalog := apps.NewCatalog(source)
	addr := os.Getenv("HOMELAB_DASHBOARD_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Homelab listening on %s", addr)
	if err := serve(ctx, listener, httpapi.NewHandler(assets, catalog), 5*time.Second); err != nil {
		log.Fatal(err)
	}
}
