package handler

import (
	"net/http"

	"couponcenter/internal/model"
	"couponcenter/pkg/httpx"
)

func (s *Server) registerTemplateRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/templates", s.createTemplate)
	mux.HandleFunc("GET /api/templates", s.listTemplates)
	mux.HandleFunc("GET /api/templates/{id}", s.getTemplate)
	mux.HandleFunc("PUT /api/templates/{id}", s.updateTemplate)
	mux.HandleFunc("DELETE /api/templates/{id}", s.deleteTemplate)
	mux.HandleFunc("POST /api/templates/{id}/transition", s.transitionTemplate)
	mux.HandleFunc("POST /api/templates/batch-archive", s.batchArchiveTemplates)
}

type createTemplateRequest struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	DiscountValue int64  `json:"discount_value"`
	MinSpend      int64  `json:"min_spend"`
	MaxDiscount   int64  `json:"max_discount"`
	TotalQuantity int    `json:"total_quantity"`
	PerUserLimit  int    `json:"per_user_limit"`
	ValidDays     int    `json:"valid_days"`
	Category      string `json:"category"`
	StartTime     string `json:"start_time"` // RFC3339，可选
	EndTime       string `json:"end_time"`   // RFC3339，可选
}

func (s *Server) createTemplate(w http.ResponseWriter, r *http.Request) {
	var req createTemplateRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	input := model.CouponTemplate{
		Code:          req.Code,
		Name:          req.Name,
		Type:          req.Type,
		DiscountValue: req.DiscountValue,
		MinSpend:      req.MinSpend,
		MaxDiscount:   req.MaxDiscount,
		TotalQuantity: req.TotalQuantity,
		PerUserLimit:  req.PerUserLimit,
		ValidDays:     req.ValidDays,
		Category:      req.Category,
	}
	if req.StartTime != "" {
		t, err := parseTimeRFC3339(req.StartTime)
		if err != nil {
			httpx.BadRequest(w, "start_time 格式不合法，需为 RFC3339")
			return
		}
		input.StartTime = t
	}
	if req.EndTime != "" {
		t, err := parseTimeRFC3339(req.EndTime)
		if err != nil {
			httpx.BadRequest(w, "end_time 格式不合法，需为 RFC3339")
			return
		}
		input.EndTime = t
	}
	tpl, err := s.svc.CreateTemplate(input)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.Created(w, tpl)
}

func (s *Server) listTemplates(w http.ResponseWriter, r *http.Request) {
	pp := httpx.ParsePagination(r, 20, s.maxPageSize())
	filter := model.TemplateFilter{
		Status:   r.URL.Query().Get("status"),
		Type:     r.URL.Query().Get("type"),
		Category: r.URL.Query().Get("category"),
		Keyword:  r.URL.Query().Get("keyword"),
	}
	items, total, err := s.svc.ListTemplates(filter, pp.Page, pp.Size)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, httpx.PageResult{
		Items:      items,
		Pagination: httpx.Pagination{Page: pp.Page, Size: pp.Size, Total: total},
	})
}

func (s *Server) getTemplate(w http.ResponseWriter, r *http.Request) {
	tpl, err := s.svc.GetTemplate(r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, tpl)
}

type updateTemplateRequest struct {
	Name         string `json:"name"`
	Category     string `json:"category"`
	MinSpend     int64  `json:"min_spend"`
	MaxDiscount  int64  `json:"max_discount"`
	PerUserLimit int    `json:"per_user_limit"`
}

func (s *Server) updateTemplate(w http.ResponseWriter, r *http.Request) {
	var req updateTemplateRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	tpl, err := s.svc.UpdateTemplate(r.PathValue("id"), req.Name, req.Category,
		req.MinSpend, req.MaxDiscount, req.PerUserLimit)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, tpl)
}

func (s *Server) deleteTemplate(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.DeleteTemplate(r.PathValue("id")); err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}

type transitionRequest struct {
	To string `json:"to"`
}

func (s *Server) transitionTemplate(w http.ResponseWriter, r *http.Request) {
	var req transitionRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	tpl, err := s.svc.TransitionTemplate(r.PathValue("id"), req.To)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, tpl)
}

type batchArchiveRequest struct {
	IDs []string `json:"ids"`
}

func (s *Server) batchArchiveTemplates(w http.ResponseWriter, r *http.Request) {
	var req batchArchiveRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	count, err := s.svc.BatchArchiveTemplates(req.IDs)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, map[string]int{"archived": count})
}
