package handler

import (
	"net/http"
	"strconv"

	"couponcenter/pkg/httpx"
)

func (s *Server) registerStatsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/stats/overview", s.statsOverview)
	mux.HandleFunc("GET /api/stats/by-template", s.statsByTemplate)
	mux.HandleFunc("GET /api/stats/by-day", s.statsByDay)
	mux.HandleFunc("GET /api/stats/top-users", s.statsTopUsers)
}

func (s *Server) statsOverview(w http.ResponseWriter, r *http.Request) {
	httpx.OK(w, s.svc.Stats())
}

func (s *Server) statsByTemplate(w http.ResponseWriter, r *http.Request) {
	httpx.OK(w, s.svc.StatsByTemplate())
}

func (s *Server) statsByDay(w http.ResponseWriter, r *http.Request) {
	httpx.OK(w, s.svc.StatsByDay())
}

func (s *Server) statsTopUsers(w http.ResponseWriter, r *http.Request) {
	n, _ := strconv.Atoi(r.URL.Query().Get("n"))
	httpx.OK(w, s.svc.TopUsers(n))
}
