package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]interface{}{
		"status":      "ok",
		"edges_count": len(s.discovery.ListEdges()),
	})
}

func (s *Server) handleListEdges(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.discovery.ListEdges())
}

func (s *Server) handleEdgeHealth(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	s.proxyToEdge(w, username, "GET", "/health", nil)
}

func (s *Server) handleEdgeRollcalls(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	s.proxyToEdge(w, username, "GET", "/rollcalls", nil)
}

func (s *Server) handleEdgePause(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "failed to read body"})
		return
	}

	var body struct {
		Pause *bool `json:"pause"`
	}
	hasPause := false
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &body); err == nil && body.Pause != nil {
			hasPause = true
		}
	}

	if hasPause {
		s.proxyToEdge(w, username, "POST", "/pause_shared", strings.NewReader(string(bodyBytes)))
	} else {
		s.proxyToEdge(w, username, "GET", "/pause_shared", nil)
	}
}

func (s *Server) handleEdgeQRCheckin(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	var body struct {
		RollcallID int64  `json:"rollcall_id"`
		Data       string `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid request body"})
		return
	}

	proxyBody, _ := json.Marshal(map[string]string{"data": body.Data})
	path := fmt.Sprintf("/rollcall/%d/qr", body.RollcallID)
	s.proxyToEdge(w, username, "POST", path, strings.NewReader(string(proxyBody)))
}

func (s *Server) handleEdgeNumberCheckin(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	var body struct {
		RollcallID int64  `json:"rollcall_id"`
		NumberCode string `json:"number_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid request body"})
		return
	}

	proxyBody, _ := json.Marshal(map[string]string{"numberCode": body.NumberCode})
	path := fmt.Sprintf("/rollcall/%d/number", body.RollcallID)
	s.proxyToEdge(w, username, "POST", path, strings.NewReader(string(proxyBody)))
}

func (s *Server) handleEdgeLocationCheckin(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	var body struct {
		RollcallID int64   `json:"rollcall_id"`
		Lat        float64 `json:"lat"`
		Lon        float64 `json:"lon"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid request body"})
		return
	}

	proxyBody, _ := json.Marshal(map[string]float64{"lat": body.Lat, "lon": body.Lon})
	path := fmt.Sprintf("/rollcall/%d/location", body.RollcallID)
	s.proxyToEdge(w, username, "POST", path, strings.NewReader(string(proxyBody)))
}

func (s *Server) handleEdgeBatchQRCheckin(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	var body struct {
		Data string `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid request body"})
		return
	}

	proxyBody, _ := json.Marshal(map[string]string{"data": body.Data})
	s.proxyToEdge(w, username, "POST", "/rollcallqr", strings.NewReader(string(proxyBody)))
}

func (s *Server) proxyToEdge(w http.ResponseWriter, username string, method string, path string, body io.Reader) {
	edge, found := s.discovery.Resolve(username)
	if !found {
		writeJSON(w, 404, map[string]string{"error": "edge not found"})
		return
	}

	targetURL := resolveHTTPAddr(edge.Addr) + path

	resp, err := proxyRequest(method, targetURL, body)
	if err != nil {
		s.log.Error("proxy request failed", "target", targetURL, "error", err)
		writeJSON(w, 502, map[string]string{"error": "edge unreachable"})
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		writeJSON(w, 502, map[string]string{"error": "failed to read edge response"})
		return
	}

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}

func proxyRequest(method, targetURL string, body io.Reader) (*http.Response, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(method, targetURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return client.Do(req)
}

func resolveHTTPAddr(addr string) string {
	addr = strings.TrimPrefix(addr, "http://")
	addr = strings.TrimPrefix(addr, "rpc://")
	return "http://" + addr
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
