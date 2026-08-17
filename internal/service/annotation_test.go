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

func TestAnnotationTemplateLifecycleClaimAndStats(t *testing.T) {
	s := annotationService()
	tpl := mustAnnotationTemplate(t, s, "WELCOME")
	listed, total, err := s.ListTemplates(model.TemplateFilter{Category: "food", Keyword: "wel"}, 1, 10)
	if err != nil {
		t.Fatalf("list templates: %v", err)
	}
	if total != 1 || len(listed) != 1 || listed[0].ID != tpl.ID {
		t.Fatalf("category/keyword list lost active template: total=%d len=%d", total, len(listed))
	}
	if _, err := s.ClaimCoupon(tpl.ID, "u-1"); err != nil {
		t.Fatalf("claim active template: %v", err)
	}
	refreshed, err := s.GetTemplate(tpl.ID)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed.IssuedCount != 1 {
		t.Fatalf("issued count = %d, want 1", refreshed.IssuedCount)
	}
	st := s.Stats()
	if st.TemplateCount != 1 || st.IssuedCount != 1 {
		t.Fatalf("stats = %+v, want one template and one issued coupon", st)
	}
}
