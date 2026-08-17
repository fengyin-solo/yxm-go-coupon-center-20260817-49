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

func TestAnnotationRedeemCodeLifecycle(t *testing.T) {
	s := annotationService()
	tpl := mustAnnotationTemplate(t, s, "RCWELCOME")
	expired, err := s.CreateRedeemCode(tpl.ID, "OLD-CODE", time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("create expired code: %v", err)
	}
	if _, err := s.RedeemByCode(expired.Code, "u-expired"); err == nil {
		t.Fatalf("expired redeem code was accepted")
	}
	rc, err := s.CreateRedeemCode(tpl.ID, "LIVE-CODE", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("create live code: %v", err)
	}
	if _, err := s.RedeemByCode("LIVE-CODE", "u-live"); err != nil {
		t.Fatalf("redeem live code: %v", err)
	}
	got, err := s.GetRedeemCode(rc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.RedeemCodeRedeemed || got.RedeemedBy != "u-live" || got.RedeemedAt.IsZero() {
		t.Fatalf("redeemed metadata not persisted: %+v", got)
	}
	list, total, err := s.ListRedeemCodes(model.RedeemCodeFilter{Status: model.RedeemCodeRedeemed, Keyword: "LIVE"}, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != rc.ID {
		t.Fatalf("redeemed code filter mismatch: total=%d list=%+v", total, list)
	}
}
