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

func TestAnnotationPaginationRejectsInvalidServiceInput(t *testing.T) {
	s := annotationService()
	mustAnnotationTemplate(t, s, "PAGE")
	if _, _, err := s.ListTemplates(model.TemplateFilter{}, 0, 10); err == nil {
		t.Fatalf("page 0 should return validation error")
	}
	if _, _, err := s.ListUserCoupons(model.UserCouponFilter{}, 1, 0); err == nil {
		t.Fatalf("size 0 should return validation error")
	}
	if _, _, err := s.ListRules(model.RuleFilter{}, -1, 10); err == nil {
		t.Fatalf("negative page should return validation error")
	}
	if _, _, err := s.ListRedeemCodes(model.RedeemCodeFilter{}, 1, -5); err == nil {
		t.Fatalf("negative size should return validation error")
	}
}
