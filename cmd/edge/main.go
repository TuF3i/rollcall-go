package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Auto-CQUPT-Plan/rollcall-go/internal/config"
	"github.com/Auto-CQUPT-Plan/rollcall-go/internal/edge"
	"github.com/Auto-CQUPT-Plan/rollcall-go/internal/lms"
	"github.com/Auto-CQUPT-Plan/rollcall-go/internal/logger"
	"github.com/Auto-CQUPT-Plan/rollcall-go/internal/notify"
	"github.com/Auto-CQUPT-Plan/rollcall-go/internal/registry"
	"github.com/Auto-CQUPT-Plan/rollcall-go/internal/rpc"
)

var (
	startupBanner = logger.Dim("━━━") + logger.Section(" Edge Server ") + logger.Dim("━━━")
)

type apiServer interface {
	Addr() string
}

type stoppableServer interface {
	Stop() error
}

type httpServerWrapper struct {
	*http.Server
}

func (h *httpServerWrapper) Addr() string {
	return h.Server.Addr
}

func (h *httpServerWrapper) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return h.Server.Shutdown(ctx)
}

func main() {
	logger.Setup(os.Stdout)

	logger.PrintBanner("banner.txt", "Edge Server")

	if err := config.Load(); err != nil {
		slog.Error("配置加载失败", "error", err)
		os.Exit(1)
	}

	slog.Info(startupBanner)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	lmsClient := lms.NewClient()
	defer lmsClient.Close()

	notify.Sendf("🟢 Edge Server 已启动\nclient_id: %s", config.ClientID[:8])
	slog.Info("正在检查 LMS 会话...")
	if _, err := lmsClient.GetRollcalls(ctx); err != nil {
		slog.Warn("初始签到检查失败（稍后重试）", "error", err)
	}

	poller := edge.NewPoller(lmsClient)
	wsClient := edge.NewWSClient(lmsClient, poller)
	poller.SetSendFunc(wsClient.SendToCenter)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("轮询器崩溃", "panic", r)
			}
		}()
		poller.Run(ctx)
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("WebSocket 客户端崩溃", "panic", r)
			}
		}()
		wsClient.Run(ctx)
	}()

	var svr stoppableServer
	var serviceAddr string

	switch config.Cfg.ApiMode {
	case "rpc":
		addr := fmt.Sprintf(":%d", config.Cfg.RPCPort)
		rpcServer, err := rpc.NewRpcServer(addr, lmsClient, wsClient, poller)
		if err != nil {
			slog.Error("RPC 服务启动失败", "error", err)
			os.Exit(1)
		}
		serviceAddr = fmt.Sprintf("rpc://%s:%d", localIP(), config.Cfg.RPCPort)
		svr = rpcServer
		go func() {
			if err := rpcServer.Run(); err != nil {
				slog.Error("RPC 服务异常", "error", err)
			}
		}()

	default:
		addr := fmt.Sprintf(":%d", *config.Cfg.HTTPPort)
		httpServer := &http.Server{
			Addr:         addr,
			Handler:      edge.NewServer(lmsClient, wsClient, poller).Router(),
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		}
		serviceAddr = fmt.Sprintf("%s:%d", localIP(), *config.Cfg.HTTPPort)
		svr = &httpServerWrapper{httpServer}
		go func() {
			slog.Info("HTTP 服务已启动", "地址", addr)
			if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
				slog.Error("HTTP 服务异常", "error", err)
			}
		}()
	}

	// Register to etcd if configured
	var registrar *registry.Registrar
	if config.Cfg.EtcdEndpoints != "" {
		r, err := registry.New(
			config.Cfg.EtcdEndpoints,
			config.Cfg.EtcdPrefix,
			"rollcall_edge",
			config.ClientID,
			config.Cfg.Username,
			serviceAddr,
		)
		if err != nil {
			slog.Error("etcd 注册失败", "error", err)
		} else {
			registrar = r
		}
	}

	sig := <-sigCh
	slog.Info("收到信号，正在关闭...", "signal", sig)
	cancel()

	if registrar != nil {
		registrar.Close()
	}
	svr.Stop()

	slog.Info("Edge Server 已停止")
}

func localIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return "127.0.0.1"
}
