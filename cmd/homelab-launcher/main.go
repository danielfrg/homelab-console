package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danielfrg/homelab-launcher/internal/apps"
	"github.com/danielfrg/homelab-launcher/internal/httpapi"
	"github.com/danielfrg/homelab-launcher/internal/web"
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
	addr := listenAddr()
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Homelab listening on %s", addr)
	if err := serve(ctx, listener, httpapi.NewHandler(assets, catalog), 5*time.Second); err != nil {
		log.Fatal(err)
	}
}

func listenAddr() string {
	if addr := os.Getenv("HOMELAB_LAUNCHER_ADDR"); addr != "" {
		return addr
	}
	return "127.0.0.1:8080"
}
