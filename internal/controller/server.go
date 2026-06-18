package controller

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	discovery *Discovery
	log       *slog.Logger
}

func NewServer(discovery *Discovery) *Server {
	return &Server{
		discovery: discovery,
		log:       slog.With("component", "controller_http"),
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	r.Get("/health", s.handleHealth)
	r.Get("/edges", s.handleListEdges)
	r.Post("/edges/{username}/health", s.handleEdgeHealth)
	r.Post("/edges/{username}/rollcalls", s.handleEdgeRollcalls)
	r.Post("/edges/{username}/pause", s.handleEdgePause)
	r.Post("/edges/{username}/qr-checkin", s.handleEdgeQRCheckin)
	r.Post("/edges/{username}/number-checkin", s.handleEdgeNumberCheckin)
	r.Post("/edges/{username}/location-checkin", s.handleEdgeLocationCheckin)
	r.Post("/edges/{username}/batch-qr-checkin", s.handleEdgeBatchQRCheckin)

	return r
}
