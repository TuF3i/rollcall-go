package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Auto-CQUPT-Plan/rollcall-go/internal/config"
	"github.com/Auto-CQUPT-Plan/rollcall-go/internal/controller"
	"github.com/Auto-CQUPT-Plan/rollcall-go/internal/logger"
)

func main() {
	logger.Setup(os.Stdout)
	logger.PrintBanner("banner.txt", "Controller")

	// Read controller config from environment (no config.Load() — avoids Username/Password requirement)
	etcdEndpoints := strings.TrimSpace(os.Getenv("EDGE_ETCD_ENDPOINTS"))
	if etcdEndpoints == "" {
		slog.Error("未配置 EDGE_ETCD_ENDPOINTS，无法发现 Edge 实例")
		os.Exit(1)
	}

	etcdPrefix := os.Getenv("EDGE_ETCD_PREFIX")
	if etcdPrefix == "" {
		etcdPrefix = "/rollcall"
	}

	httpPort := 8082
	if v := os.Getenv("CONTROLLER_HTTP_PORT"); v != "" {
		var p int
		if _, err := fmt.Sscanf(v, "%d", &p); err == nil {
			httpPort = p
		}
	}

	// Set shared config fields used by controller.NewDiscovery
	config.Cfg.EtcdEndpoints = etcdEndpoints
	config.Cfg.EtcdPrefix = etcdPrefix
	config.Cfg.ControllerHTTPPort = httpPort

	slog.Info("━━━ Controller ━━━")
	slog.Info("正在连接 etcd", "地址", etcdEndpoints)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	disc, err := controller.NewDiscovery(etcdEndpoints)
	if err != nil {
		slog.Error("etcd 连接失败", "error", err)
		os.Exit(1)
	}
	defer disc.Close()

	// Run is non-blocking (starts watch goroutine internally)
	disc.Run(ctx)

	addr := fmt.Sprintf(":%d", httpPort)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      controller.NewServer(disc).Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("HTTP 服务已启动", "地址", addr)
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("HTTP 服务异常", "error", err)
			os.Exit(1)
		}
	}()

	sig := <-sigCh
	slog.Info("收到信号，正在关闭...", "signal", sig)
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	httpServer.Shutdown(shutdownCtx)

	slog.Info("Controller 已停止")
}
