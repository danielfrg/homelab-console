package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/danielfrg/homelab-launcher/internal/apps"
	"github.com/danielfrg/homelab-launcher/internal/cluster"
	"github.com/danielfrg/homelab-launcher/internal/httpapi"
	"github.com/danielfrg/homelab-launcher/internal/mcp"
	"github.com/danielfrg/homelab-launcher/internal/web"
	"k8s.io/client-go/kubernetes"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	hosts, err := mcp.ParseAllowedHosts(os.Getenv("HOMELAB_LAUNCHER_MCP_ALLOWED_HOSTS"))
	if err != nil {
		log.Fatal(err)
	}
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
	reader, err := cluster.NewKubeReader(config)
	if err != nil {
		log.Fatal(err)
	}
	typed, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	enabled := restartEnabled(os.Getenv("HOMELAB_CONSOLE_RESTART_ENABLED"))
	console := cluster.NewConsole(cluster.NewInventory(reader, catalog), typed, enabled, time.Now)
	log.Printf("Restart enabled: %v", enabled)
	addr := listenAddr()
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Homelab Console listening on %s", addr)
	if err := serve(ctx, listener, httpapi.NewHandler(assets, catalog, console, hosts...), 5*time.Second); err != nil {
		log.Fatal(err)
	}
}

func listenAddr() string {
	if addr := os.Getenv("HOMELAB_LAUNCHER_ADDR"); addr != "" {
		return addr
	}
	return "127.0.0.1:8080"
}

// restartEnabled defaults to on; only an explicit "false" turns restart off.
func restartEnabled(value string) bool {
	return !strings.EqualFold(strings.TrimSpace(value), "false")
}
