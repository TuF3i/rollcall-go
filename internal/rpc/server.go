package rpc

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/Auto-CQUPT-Plan/rollcall-go/internal/edge"
	"github.com/Auto-CQUPT-Plan/rollcall-go/internal/lms"
	"github.com/Auto-CQUPT-Plan/rollcall-go/internal/logger"
	"github.com/Auto-CQUPT-Plan/rollcall-go/internal/rpc/kitex_gen/edge/edgeservice"
	"github.com/cloudwego/kitex/server"
)

type Server struct {
	svr  server.Server
	addr string
}

func NewRpcServer(addr string, lmsClient *lms.Client, wsClient *edge.WSClient, poller *edge.Poller) (*Server, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("RPC 地址解析失败: %w", err)
	}

	handler := NewEdgeServiceImpl(lmsClient, wsClient, poller)
	svr := edgeservice.NewServer(
		handler,
		server.WithServiceAddr(tcpAddr),
	)

	return &Server{
		svr:  svr,
		addr: tcpAddr.String(),
	}, nil
}

func (s *Server) Run() error {
	slog.Info(fmt.Sprintf("%s %s",
		logger.TagOK("RPC 服务已启动"),
		logger.KV("地址", s.addr)))
	return s.svr.Run()
}

func (s *Server) Stop() error {
	return s.svr.Stop()
}

func (s *Server) Addr() string {
	return s.addr
}
