package handler

import (
	"net/http"
	"time"

	"couponcenter/internal/model"
	"couponcenter/pkg/httpx"
)

func (s *Server) registerRedeemCodeRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/redeem-codes", s.createRedeemCode)
	mux.HandleFunc("POST /api/redeem-codes/batch", s.batchCreateRedeemCodes)
	mux.HandleFunc("GET /api/redeem-codes", s.listRedeemCodes)
	mux.HandleFunc("GET /api/redeem-codes/{id}", s.getRedeemCode)
	mux.HandleFunc("POST /api/redeem-codes/redeem", s.redeemByCode)
	mux.HandleFunc("POST /api/redeem-codes/{id}/void", s.voidRedeemCode)
	mux.HandleFunc("DELETE /api/redeem-codes/{id}", s.deleteRedeemCode)
}

type createRedeemCodeRequest struct {
	TemplateID string `json:"template_id"`
	Code       string `json:"code"`
	ExpireAt   string `json:"expire_at"` // RFC3339，可选
}

func (s *Server) createRedeemCode(w http.ResponseWriter, r *http.Request) {
	var req createRedeemCodeRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	var expireAt time.Time
	if req.ExpireAt != "" {
		t, err := parseTimeRFC3339(req.ExpireAt)
		if err != nil {
			httpx.BadRequest(w, "expire_at 格式不合法，需为 RFC3339")
			return
		}
		expireAt = t
	}
	rc, err := s.svc.CreateRedeemCode(req.TemplateID, req.Code, expireAt)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.Created(w, rc)
}

type batchCreateRedeemCodesRequest struct {
	TemplateID string `json:"template_id"`
	Count      int    `json:"count"`
	ExpireAt   string `json:"expire_at"` // RFC3339，可选
}

func (s *Server) batchCreateRedeemCodes(w http.ResponseWriter, r *http.Request) {
	var req batchCreateRedeemCodesRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	var expireAt time.Time
	if req.ExpireAt != "" {
		t, err := parseTimeRFC3339(req.ExpireAt)
		if err != nil {
			httpx.BadRequest(w, "expire_at 格式不合法，需为 RFC3339")
			return
		}
		expireAt = t
	}
	count, err := s.svc.BatchCreateRedeemCodes(req.TemplateID, req.Count, expireAt)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.Created(w, map[string]int{"created": count})
}

func (s *Server) listRedeemCodes(w http.ResponseWriter, r *http.Request) {
	pp := httpx.ParsePagination(r, 20, s.maxPageSize())
	filter := model.RedeemCodeFilter{
		TemplateID: r.URL.Query().Get("template_id"),
		Status:     r.URL.Query().Get("status"),
		Keyword:    r.URL.Query().Get("keyword"),
	}
	items, total, err := s.svc.ListRedeemCodes(filter, pp.Page, pp.Size)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, httpx.PageResult{
		Items:      items,
		Pagination: httpx.Pagination{Page: pp.Page, Size: pp.Size, Total: total},
	})
}

func (s *Server) getRedeemCode(w http.ResponseWriter, r *http.Request) {
	rc, err := s.svc.GetRedeemCode(r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, rc)
}

type redeemByCodeRequest struct {
	Code   string `json:"code"`
	UserID string `json:"user_id"`
}

func (s *Server) redeemByCode(w http.ResponseWriter, r *http.Request) {
	var req redeemByCodeRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	coupon, err := s.svc.RedeemByCode(req.Code, req.UserID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.Created(w, coupon)
}

func (s *Server) voidRedeemCode(w http.ResponseWriter, r *http.Request) {
	rc, err := s.svc.VoidRedeemCode(r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, rc)
}

func (s *Server) deleteRedeemCode(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.DeleteRedeemCode(r.PathValue("id")); err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.NoContent(w)
}
