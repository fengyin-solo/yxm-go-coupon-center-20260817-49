package handler

import (
	"net/http"

	"couponcenter/internal/model"
	"couponcenter/pkg/httpx"
)

func (s *Server) registerRuleRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/rules", s.createRule)
	mux.HandleFunc("GET /api/rules", s.listRules)
	mux.HandleFunc("GET /api/rules/{id}", s.getRule)
	mux.HandleFunc("PUT /api/rules/{id}", s.updateRule)
	mux.HandleFunc("DELETE /api/rules/{id}", s.deleteRule)
}

type createRuleRequest struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	TemplateID string `json:"template_id"`
	LimitValue int64  `json:"limit_value"`
	TextValue  string `json:"text_value"`
}

func (s *Server) createRule(w http.ResponseWriter, r *http.Request) {
	var req createRuleRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	rule, err := s.svc.CreateRule(model.Rule{
		Name:       req.Name,
		Type:       req.Type,
		TemplateID: req.TemplateID,
		LimitValue: req.LimitValue,
		TextValue:  req.TextValue,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.Created(w, rule)
}

func (s *Server) listRules(w http.ResponseWriter, r *http.Request) {
	pp := httpx.ParsePagination(r, 20, s.maxPageSize())
	filter := model.RuleFilter{
		Type:       r.URL.Query().Get("type"),
		Status:     r.URL.Query().Get("status"),
		TemplateID: r.URL.Query().Get("template_id"),
	}
	items, total, err := s.svc.ListRules(filter, pp.Page, pp.Size)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, httpx.PageResult{
		Items:      items,
		Pagination: httpx.Pagination{Page: pp.Page, Size: pp.Size, Total: total},
	})
}

func (s *Server) getRule(w http.ResponseWriter, r *http.Request) {
	rule, err := s.svc.GetRule(r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, rule)
}

type updateRuleRequest struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	LimitValue int64  `json:"limit_value"`
	TextValue  string `json:"text_value"`
}

func (s *Server) updateRule(w http.ResponseWriter, r *http.Request) {
	var req updateRuleRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	rule, err := s.svc.UpdateRule(r.PathValue("id"), req.Name, req.Status, req.LimitValue, req.TextValue)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, rule)
}

func (s *Server) deleteRule(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.DeleteRule(r.PathValue("id")); err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}
