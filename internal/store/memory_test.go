package store

import (
	"testing"
	"time"

	"couponcenter/internal/model"
)

func newTemplate(id, code string) *model.CouponTemplate {
	return &model.CouponTemplate{
		ID:            id,
		Code:          code,
		Name:          "测试券",
		Type:          model.TypeFixed,
		DiscountValue: 500,
		Status:        model.TemplateActive,
		CreatedAt:     time.Now(),
	}
}

func TestTemplateCRUD(t *testing.T) {
	s := NewMemoryStore()
	tpl := newTemplate("t1", "NEWUSER")

	if err := s.CreateTemplate(tpl); err != nil {
		t.Fatalf("创建模板失败: %v", err)
	}
	got, err := s.GetTemplate("t1")
	if err != nil || got.Code != "NEWUSER" {
		t.Fatalf("查询模板失败: %v", err)
	}
	if _, err := s.GetTemplateByCode("NEWUSER"); err != nil {
		t.Fatalf("按编码查询失败: %v", err)
	}
	if _, err := s.GetTemplateByCode("NOPE"); err != ErrNotFound {
		t.Fatalf("期望 ErrNotFound，得到 %v", err)
	}
	if list := s.ListTemplates(); len(list) != 1 {
		t.Fatalf("期望 1 个模板，得到 %d", len(list))
	}
	tpl.Name = "新名字"
	if err := s.UpdateTemplate(tpl); err != nil {
		t.Fatalf("更新模板失败: %v", err)
	}
	if err := s.DeleteTemplate("t1"); err != nil {
		t.Fatalf("删除模板失败: %v", err)
	}
	if _, err := s.GetTemplate("t1"); err != ErrNotFound {
		t.Fatalf("删除后期望 ErrNotFound，得到 %v", err)
	}
}

func TestTemplateCodeConflict(t *testing.T) {
	s := NewMemoryStore()
	if err := s.CreateTemplate(newTemplate("t1", "DUP")); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateTemplate(newTemplate("t2", "DUP")); err != ErrConflict {
		t.Fatalf("期望 ErrConflict，得到 %v", err)
	}
	// 更新时改成其他模板的编码也应冲突
	other := newTemplate("t3", "OTHER")
	_ = s.CreateTemplate(other)
	other.Code = "DUP"
	if err := s.UpdateTemplate(other); err != ErrConflict {
		t.Fatalf("更新编码冲突期望 ErrConflict，得到 %v", err)
	}
}

func TestTemplateNotFoundOps(t *testing.T) {
	s := NewMemoryStore()
	if _, err := s.GetTemplate("missing"); err != ErrNotFound {
		t.Fatalf("期望 ErrNotFound，得到 %v", err)
	}
	if err := s.UpdateTemplate(newTemplate("missing", "X")); err != ErrNotFound {
		t.Fatalf("期望 ErrNotFound，得到 %v", err)
	}
	if err := s.DeleteTemplate("missing"); err != ErrNotFound {
		t.Fatalf("期望 ErrNotFound，得到 %v", err)
	}
}

func newUserCoupon(id, templateID, userID string) *model.UserCoupon {
	return &model.UserCoupon{
		ID:         id,
		TemplateID: templateID,
		UserID:     userID,
		Type:       model.TypeFixed,
		Discount:   500,
		Status:     model.UserCouponUnused,
		ExpireAt:   time.Now().Add(24 * time.Hour),
		ReceivedAt: time.Now(),
	}
}

func TestUserCouponCRUD(t *testing.T) {
	s := NewMemoryStore()
	c := newUserCoupon("c1", "t1", "u1")
	if err := s.CreateUserCoupon(c); err != nil {
		t.Fatal(err)
	}
	if got, err := s.GetUserCoupon("c1"); err != nil || got.UserID != "u1" {
		t.Fatalf("查询用户券失败: %v", err)
	}
	if list := s.ListUserCoupons(); len(list) != 1 {
		t.Fatalf("期望 1 张券，得到 %d", len(list))
	}
	c.Status = model.UserCouponLocked
	if err := s.UpdateUserCoupon(c); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteUserCoupon("c1"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetUserCoupon("c1"); err != ErrNotFound {
		t.Fatalf("删除后期望 ErrNotFound，得到 %v", err)
	}
	if err := s.DeleteUserCoupon("c1"); err != ErrNotFound {
		t.Fatalf("重复删除期望 ErrNotFound，得到 %v", err)
	}
}

func TestCountUserCouponsByTemplate(t *testing.T) {
	s := NewMemoryStore()
	_ = s.CreateUserCoupon(newUserCoupon("c1", "t1", "u1"))
	_ = s.CreateUserCoupon(newUserCoupon("c2", "t1", "u1"))
	_ = s.CreateUserCoupon(newUserCoupon("c3", "t2", "u1"))
	_ = s.CreateUserCoupon(newUserCoupon("c4", "t1", "u2"))

	if got := s.CountUserCouponsByTemplate("u1", "t1"); got != 2 {
		t.Fatalf("期望 2，得到 %d", got)
	}
	if got := s.CountUserCouponsByTemplate("u2", "t1"); got != 1 {
		t.Fatalf("期望 1，得到 %d", got)
	}
	if got := s.CountUserCouponsByTemplate("u1", "t9"); got != 0 {
		t.Fatalf("期望 0，得到 %d", got)
	}
}

func newUsage(id, couponID string) *model.UsageRecord {
	return &model.UsageRecord{
		ID:          id,
		CouponID:    couponID,
		TemplateID:  "t1",
		UserID:      "u1",
		OrderID:     "o-" + id,
		OrderAmount: 10000,
		DiscountAmt: 500,
		UsedAt:      time.Now(),
	}
}

func TestUsageRecordCreateAndConflict(t *testing.T) {
	s := NewMemoryStore()
	if err := s.CreateUsageRecord(newUsage("u1", "c1")); err != nil {
		t.Fatal(err)
	}
	// 同一张券不能重复核销
	if err := s.CreateUsageRecord(newUsage("u2", "c1")); err != ErrConflict {
		t.Fatalf("期望 ErrConflict，得到 %v", err)
	}
	if got, err := s.GetUsageRecord("u1"); err != nil || got.OrderAmount != 10000 {
		t.Fatalf("查询使用记录失败: %v", err)
	}
	if _, err := s.GetUsageRecord("missing"); err != ErrNotFound {
		t.Fatalf("期望 ErrNotFound，得到 %v", err)
	}
	if list := s.ListUsageRecords(); len(list) != 1 {
		t.Fatalf("期望 1 条记录，得到 %d", len(list))
	}
}

func newRule(id, name, ruleType string) *model.Rule {
	return &model.Rule{
		ID:         id,
		Name:       name,
		Type:       ruleType,
		LimitValue: 3,
		Status:     model.RuleActive,
		CreatedAt:  time.Now(),
	}
}

func TestRuleCRUD(t *testing.T) {
	s := NewMemoryStore()
	r := newRule("r1", "每日限核销", model.RuleDailyUserLimit)
	if err := s.CreateRule(r); err != nil {
		t.Fatal(err)
	}
	if got, err := s.GetRule("r1"); err != nil || got.Name != "每日限核销" {
		t.Fatalf("查询规则失败: %v", err)
	}
	if list := s.ListRules(); len(list) != 1 {
		t.Fatalf("期望 1 条规则，得到 %d", len(list))
	}
	r.Status = model.RuleInactive
	if err := s.UpdateRule(r); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteRule("r1"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetRule("r1"); err != ErrNotFound {
		t.Fatalf("删除后期望 ErrNotFound，得到 %v", err)
	}
	if err := s.UpdateRule(newRule("missing", "x", model.RuleOrderCap)); err != ErrNotFound {
		t.Fatalf("期望 ErrNotFound，得到 %v", err)
	}
}

func newRedeemCode(id, code, templateID string) *model.RedemptionCode {
	return &model.RedemptionCode{
		ID:         id,
		Code:       code,
		TemplateID: templateID,
		Status:     model.RedeemCodeAvailable,
		CreatedAt:  time.Now(),
	}
}

func TestRedeemCodeCRUDAndConflict(t *testing.T) {
	s := NewMemoryStore()
	rc := newRedeemCode("rc1", "ABC123", "t1")
	if err := s.CreateRedeemCode(rc); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateRedeemCode(newRedeemCode("rc2", "ABC123", "t1")); err != ErrConflict {
		t.Fatalf("重复码期望 ErrConflict，得到 %v", err)
	}
	if got, err := s.GetRedeemCodeByCode("ABC123"); err != nil || got.ID != "rc1" {
		t.Fatalf("按码查询失败: %v", err)
	}
	if _, err := s.GetRedeemCodeByCode("NOPE"); err != ErrNotFound {
		t.Fatalf("期望 ErrNotFound，得到 %v", err)
	}
	if list := s.ListRedeemCodes(); len(list) != 1 {
		t.Fatalf("期望 1 个兑换码，得到 %d", len(list))
	}
	rc.Status = model.RedeemCodeRedeemed
	if err := s.UpdateRedeemCode(rc); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteRedeemCode("rc1"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetRedeemCode("rc1"); err != ErrNotFound {
		t.Fatalf("删除后期望 ErrNotFound，得到 %v", err)
	}
}
