package service

import (
	"testing"
	"time"

	"couponcenter/internal/config"
	"couponcenter/internal/model"
	"couponcenter/internal/store"
	"couponcenter/pkg/logger"
)

func annotationService() *Service {
	return New(store.NewMemoryStore(), logger.NewLevel(logger.LevelError), &config.Config{MaxPageSize: 100})
}

func mustAnnotationTemplate(t *testing.T, s *Service, code string) *model.CouponTemplate {
	t.Helper()
	tpl, err := s.CreateTemplate(model.CouponTemplate{
		Code: code, Name: code + " name", Type: model.TypePercent,
		DiscountValue: 80, MinSpend: 1000, MaxDiscount: 1500,
		TotalQuantity: 10, PerUserLimit: 2, ValidDays: 7, Category: "food", StartTime: time.Now().Add(-time.Hour),
	})
	if err != nil {
		t.Fatalf("create template: %v", err)
	}
	tpl, err = s.TransitionTemplate(tpl.ID, model.TemplateActive)
	if err != nil {
		t.Fatalf("activate template: %v", err)
	}
	return tpl
}

func TestAnnotationUsageRulesAndStats(t *testing.T) {
	s := annotationService()
	tpl := mustAnnotationTemplate(t, s, "RULED")
	if _, err := s.CreateRule(model.Rule{Name: "global cap", Type: model.RuleOrderCap, LimitValue: 500}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateRule(model.Rule{Name: "no toys", Type: model.RuleCategoryExclude, TextValue: "toys"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateRule(model.Rule{Name: "daily one", Type: model.RuleDailyUserLimit, TemplateID: tpl.ID, LimitValue: 1}); err != nil {
		t.Fatal(err)
	}
	blocked, err := s.ClaimCoupon(tpl.ID, "u-rule")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.LockCoupon(blocked.ID, "order-blocked"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RedeemCoupon(blocked.ID, "order-blocked", 10000, "toys"); err == nil {
		t.Fatalf("excluded category was allowed")
	}
	c, err := s.ClaimCoupon(tpl.ID, "u-rule")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.LockCoupon(c.ID, "order-ok"); err != nil {
		t.Fatal(err)
	}
	rec, err := s.RedeemCoupon(c.ID, "order-ok", 10000, "books")
	if err != nil {
		t.Fatalf("redeem with allowed category: %v", err)
	}
	if rec.DiscountAmt != 500 {
		t.Fatalf("discount = %d, want global cap 500", rec.DiscountAmt)
	}
	st := s.Stats()
	if st.TotalDiscount != 500 || st.TotalOrderAmt != 10000 || st.UsedCount != 1 {
		t.Fatalf("stats after redeem = %+v", st)
	}
}
