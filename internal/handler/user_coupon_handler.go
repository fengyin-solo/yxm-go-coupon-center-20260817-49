package handler

import (
	"net/http"

	"couponcenter/internal/model"
	"couponcenter/pkg/httpx"
)

func (s *Server) registerUserCouponRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/coupons/claim", s.claimCoupon)
	mux.HandleFunc("GET /api/coupons", s.listUserCoupons)
	mux.HandleFunc("GET /api/coupons/{id}", s.getUserCoupon)
	mux.HandleFunc("POST /api/coupons/{id}/lock", s.lockCoupon)
	mux.HandleFunc("POST /api/coupons/{id}/unlock", s.unlockCoupon)
	mux.HandleFunc("POST /api/coupons/{id}/redeem", s.redeemCoupon)
	mux.HandleFunc("POST /api/coupons/expire-scan", s.expireScan)
}

type claimCouponRequest struct {
	TemplateID string `json:"template_id"`
	UserID     string `json:"user_id"`
}

func (s *Server) claimCoupon(w http.ResponseWriter, r *http.Request) {
	var req claimCouponRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	coupon, err := s.svc.ClaimCoupon(req.TemplateID, req.UserID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.Created(w, coupon)
}

func (s *Server) listUserCoupons(w http.ResponseWriter, r *http.Request) {
	pp := httpx.ParsePagination(r, 20, s.maxPageSize())
	filter := model.UserCouponFilter{
		UserID:     r.URL.Query().Get("user_id"),
		TemplateID: r.URL.Query().Get("template_id"),
		Status:     r.URL.Query().Get("status"),
	}
	items, total, err := s.svc.ListUserCoupons(filter, pp.Page, pp.Size)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, httpx.PageResult{
		Items:      items,
		Pagination: httpx.Pagination{Page: pp.Page, Size: pp.Size, Total: total},
	})
}

func (s *Server) getUserCoupon(w http.ResponseWriter, r *http.Request) {
	coupon, err := s.svc.GetUserCoupon(r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, coupon)
}

type lockCouponRequest struct {
	OrderID string `json:"order_id"`
}

func (s *Server) lockCoupon(w http.ResponseWriter, r *http.Request) {
	var req lockCouponRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	coupon, err := s.svc.LockCoupon(r.PathValue("id"), req.OrderID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, coupon)
}

func (s *Server) unlockCoupon(w http.ResponseWriter, r *http.Request) {
	coupon, err := s.svc.UnlockCoupon(r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.OK(w, coupon)
}

type redeemCouponRequest struct {
	OrderID     string `json:"order_id"`
	OrderAmount int64  `json:"order_amount"` // 单位：分
	Category    string `json:"category"`
}

func (s *Server) redeemCoupon(w http.ResponseWriter, r *http.Request) {
	var req redeemCouponRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.BadRequest(w, "请求体解析失败: "+err.Error())
		return
	}
	record, err := s.svc.RedeemCoupon(r.PathValue("id"), req.OrderID, req.OrderAmount, req.Category)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpx.Created(w, record)
}

func (s *Server) expireScan(w http.ResponseWriter, r *http.Request) {
	count := s.svc.ExpireCoupons(timeNow())
	httpx.OK(w, map[string]int{"expired": count})
}
