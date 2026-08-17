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

func TestAnnotationStoreSnapshotsAreImmutable(t *testing.T) {
	st := store.NewMemoryStore()
	tpl := &model.CouponTemplate{ID: "t1", Code: "IMM", Name: "immutable", Type: model.TypeFixed, DiscountValue: 100, Status: model.TemplateActive, CreatedAt: time.Now()}
	if err := st.CreateTemplate(tpl); err != nil {
		t.Fatal(err)
	}
	gotTpl, _ := st.GetTemplate("t1")
	gotTpl.Code = "BROKEN"
	againTpl, _ := st.GetTemplate("t1")
	if againTpl.Code != "IMM" {
		t.Fatalf("template pointer mutation leaked into store: %q", againTpl.Code)
	}
	c := &model.UserCoupon{ID: "c1", TemplateID: "t1", UserID: "u1", Type: model.TypeFixed, Discount: 100, ExpireAt: time.Now().Add(time.Hour), Status: model.UserCouponUnused, ReceivedAt: time.Now()}
	if err := st.CreateUserCoupon(c); err != nil {
		t.Fatal(err)
	}
	listed := st.ListUserCoupons()
	listed[0].Status = model.UserCouponUsed
	againCoupon, _ := st.GetUserCoupon("c1")
	if againCoupon.Status != model.UserCouponUnused {
		t.Fatalf("coupon list mutation leaked into store: %q", againCoupon.Status)
	}
}
