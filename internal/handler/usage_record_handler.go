package handler

import (
	"net/http"

	"couponcenter/internal/model"
	"couponcenter/pkg/httpx"
)

func (s *Server) registerUsageRecordRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/usage-records", s.listUsageRecords)
	mux.HandleFunc("GET /api/usage-records/{id}", s.getUsageRecord)
}

func (s *Server) listUsageRecords(w http.ResponseWriter, r *http.Request) {
	pp := httpx.ParsePagination(r, 20, s.maxPageSize())
	filter := model.UsageFilter{
		UserID:     r.URL.Query().Get("user_id"),
		TemplateID: r.URL.Query().Get("template_id"),
		OrderID:    r.URL.Query().Get("order_id"),
	}
	if v := r.URL.Query().Get("from"); v != "" {
		t, err := parseTimeRFC3339(v)
		if err != nil {
			httpx.BadRequest(w, "from 格式不合法，需为 RFC3339")
			return
		}
		filter.From = t
	}
	if v := r.URL.Query().Get("to"); v != "" {
		t, err := parseTimeRFC3339(v)
		if err != nil {
			httpx.BadRequest(w, "to 格式不合法，需为 RFC3339")
			return
		}
		filter.To = t
	}
	items, total, err := s.svc.ListUsageRecords(filter, pp.Page, pp.Size)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, httpx.PageResult{
		Items:      items,
		Pagination: httpx.Pagination{Page: pp.Page, Size: pp.Size, Total: total},
	})
}

func (s *Server) getUsageRecord(w http.ResponseWriter, r *http.Request) {
	record, err := s.svc.GetUsageRecord(r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, record)
}
