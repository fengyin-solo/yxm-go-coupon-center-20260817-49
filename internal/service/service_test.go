package service

import (
	"testing"
	"time"

	"couponcenter/internal/config"
	"couponcenter/internal/model"
	"couponcenter/internal/store"
	"couponcenter/pkg/logger"
)

func newTestService() *Service {
	cfg := &config.Config{MaxPageSize: 100}
	log := logger.NewLevel(logger.LevelError)
	return New(store.NewMemoryStore(), log, cfg)
}

// activeFixedTemplate 创建并激活一个满减券模板，返回模板。
func activeFixedTemplate(t *testing.T, svc *Service, code string) *model.CouponTemplate {
	t.Helper()
	tpl, err := svc.CreateTemplate(model.CouponTemplate{
		Code:          code,
		Name:          "满 100 减 5 元",
		Type:          model.TypeFixed,
		DiscountValue: 500,
		MinSpend:      10000,
		TotalQuantity: 100,
		PerUserLimit:  2,
		ValidDays:     7,
	})
	if err != nil {
		t.Fatalf("创建模板失败: %v", err)
	}
	tpl, err = svc.TransitionTemplate(tpl.ID, model.TemplateActive)
	if err != nil {
		t.Fatalf("激活模板失败: %v", err)
	}
	return tpl
}

func TestCreateTemplateValidation(t *testing.T) {
	svc := newTestService()
	cases := []struct {
		name  string
		input model.CouponTemplate
	}{
		{"空编码", model.CouponTemplate{Name: "x", Type: model.TypeFixed, DiscountValue: 100}},
		{"空名称", model.CouponTemplate{Code: "C1", Type: model.TypeFixed, DiscountValue: 100}},
		{"非法类型", model.CouponTemplate{Code: "C1", Name: "x", Type: "weird", DiscountValue: 100}},
		{"满减金额为零", model.CouponTemplate{Code: "C1", Name: "x", Type: model.TypeFixed, DiscountValue: 0}},
		{"折扣超范围", model.CouponTemplate{Code: "C1", Name: "x", Type: model.TypePercent, DiscountValue: 120}},
		{"负门槛", model.CouponTemplate{Code: "C1", Name: "x", Type: model.TypeFixed, DiscountValue: 100, MinSpend: -1}},
	}
	for _, c := range cases {
		if _, err := svc.CreateTemplate(c.input); err == nil {
			t.Errorf("%s: 期望校验失败但成功了", c.name)
		} else if !model.IsValidationError(err) {
			t.Errorf("%s: 期望 ValidationError，得到 %v", c.name, err)
		}
	}
}

func TestTemplateLifecycle(t *testing.T) {
	svc := newTestService()
	tpl, err := svc.CreateTemplate(model.CouponTemplate{
		Code: "LIFE", Name: "生命周期", Type: model.TypeFixed, DiscountValue: 300,
	})
	if err != nil {
		t.Fatal(err)
	}
	if tpl.Status != model.TemplateDraft {
		t.Fatalf("新模板应为 draft，得到 %s", tpl.Status)
	}
	// 非法流转：draft -> paused
	if _, err := svc.TransitionTemplate(tpl.ID, model.TemplatePaused); err == nil {
		t.Fatal("draft->paused 应被拒绝")
	}
	// draft -> active -> paused -> active -> archived
	for _, to := range []string{model.TemplateActive, model.TemplatePaused, model.TemplateActive, model.TemplateArchived} {
		tpl, err = svc.TransitionTemplate(tpl.ID, to)
		if err != nil {
			t.Fatalf("流转到 %s 失败: %v", to, err)
		}
		if tpl.Status != to {
			t.Fatalf("期望状态 %s，得到 %s", to, tpl.Status)
		}
	}
	// 归档后不可编辑
	if _, err := svc.UpdateTemplate(tpl.ID, "新名字", "", -1, -1, -1); err == nil {
		t.Fatal("归档模板编辑应被拒绝")
	}
}

func TestTemplateListFilterAndPagination(t *testing.T) {
	svc := newTestService()
	for i, code := range []string{"A1", "A2", "B1"} {
		tpl, err := svc.CreateTemplate(model.CouponTemplate{
			Code: code, Name: "券" + code, Type: model.TypeFixed, DiscountValue: int64(100 + i),
		})
		if err != nil {
			t.Fatal(err)
		}
		if i < 2 {
			if _, err := svc.TransitionTemplate(tpl.ID, model.TemplateActive); err != nil {
				t.Fatal(err)
			}
		}
	}
	items, total, err := svc.ListTemplates(model.TemplateFilter{Status: model.TemplateActive}, 1, 10)
	if err != nil || total != 2 || len(items) != 2 {
		t.Fatalf("状态筛选期望 2 条，得到 total=%d len=%d err=%v", total, len(items), err)
	}
	items, total, err = svc.ListTemplates(model.TemplateFilter{Keyword: "b1"}, 1, 10)
	if err != nil || total != 1 || items[0].Code != "B1" {
		t.Fatalf("关键词筛选失败: total=%d err=%v", total, err)
	}
	// 分页：每页 2 条，第 2 页应剩 1 条
	items, total, err = svc.ListTemplates(model.TemplateFilter{}, 2, 2)
	if err != nil || total != 3 || len(items) != 1 {
		t.Fatalf("分页期望 total=3 len=1，得到 total=%d len=%d", total, len(items))
	}
}

func TestDeleteTemplateWithCoupons(t *testing.T) {
	svc := newTestService()
	tpl := activeFixedTemplate(t, svc, "DEL1")
	if _, err := svc.ClaimCoupon(tpl.ID, "u1"); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteTemplate(tpl.ID); err == nil {
		t.Fatal("已有领券记录的模板删除应被拒绝")
	}
}

func TestBatchArchiveTemplates(t *testing.T) {
	svc := newTestService()
	t1 := activeFixedTemplate(t, svc, "BA1")
	t2 := activeFixedTemplate(t, svc, "BA2")
	// 草稿模板不应被归档
	draft, _ := svc.CreateTemplate(model.CouponTemplate{
		Code: "BA3", Name: "草稿", Type: model.TypeFixed, DiscountValue: 100,
	})
	count, err := svc.BatchArchiveTemplates([]string{t1.ID, t2.ID, draft.ID, "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("期望归档 2 个，得到 %d", count)
	}
	if _, err := svc.BatchArchiveTemplates(nil); err == nil {
		t.Fatal("空列表应报错")
	}
}

func TestClaimCouponHappyPath(t *testing.T) {
	svc := newTestService()
	tpl := activeFixedTemplate(t, svc, "CLAIM1")
	coupon, err := svc.ClaimCoupon(tpl.ID, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if coupon.Status != model.UserCouponUnused || coupon.Type != model.TypeFixed {
		t.Fatalf("券状态/类型不对: %s %s", coupon.Status, coupon.Type)
	}
	if coupon.ExpireAt.Before(time.Now()) {
		t.Fatal("过期时间不应早于当前")
	}
	got, err := svc.GetTemplate(tpl.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.IssuedCount != 1 {
		t.Fatalf("发放数应为 1，得到 %d", got.IssuedCount)
	}
}

func TestClaimCouponRestrictions(t *testing.T) {
	svc := newTestService()
	tpl := activeFixedTemplate(t, svc, "CLAIM2") // PerUserLimit=2
	if _, err := svc.ClaimCoupon(tpl.ID, "u1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ClaimCoupon(tpl.ID, "u1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ClaimCoupon(tpl.ID, "u1"); err == nil {
		t.Fatal("超过每人限领应被拒绝")
	}
	// 草稿模板不可领取
	draft, _ := svc.CreateTemplate(model.CouponTemplate{
		Code: "CLAIM3", Name: "草稿", Type: model.TypeFixed, DiscountValue: 100,
	})
	if _, err := svc.ClaimCoupon(draft.ID, "u1"); err == nil {
		t.Fatal("草稿模板领取应被拒绝")
	}
	// 不存在的模板
	if _, err := svc.ClaimCoupon("missing", "u1"); err == nil {
		t.Fatal("不存在的模板领取应被拒绝")
	}
}

func TestClaimCouponOutOfStock(t *testing.T) {
	svc := newTestService()
	tpl, err := svc.CreateTemplate(model.CouponTemplate{
		Code: "STOCK", Name: "限量", Type: model.TypeFixed, DiscountValue: 100,
		TotalQuantity: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.TransitionTemplate(tpl.ID, model.TemplateActive); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ClaimCoupon(tpl.ID, "u1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ClaimCoupon(tpl.ID, "u2"); err == nil {
		t.Fatal("库存耗尽应被拒绝")
	}
}

// claimAndLock 领取并锁定一张券，返回券。
func claimAndLock(t *testing.T, svc *Service, tplID, userID, orderID string) *model.UserCoupon {
	t.Helper()
	coupon, err := svc.ClaimCoupon(tplID, userID)
	if err != nil {
		t.Fatal(err)
	}
	coupon, err = svc.LockCoupon(coupon.ID, orderID)
	if err != nil {
		t.Fatal(err)
	}
	return coupon
}

func TestLockUnlockRedeemFlow(t *testing.T) {
	svc := newTestService()
	tpl := activeFixedTemplate(t, svc, "FLOW1")
	coupon := claimAndLock(t, svc, tpl.ID, "u1", "order-1")
	if coupon.Status != model.UserCouponLocked || coupon.OrderID != "order-1" {
		t.Fatalf("锁定后状态不对: %s %s", coupon.Status, coupon.OrderID)
	}
	// 解锁
	coupon, err := svc.UnlockCoupon(coupon.ID)
	if err != nil || coupon.Status != model.UserCouponUnused || coupon.OrderID != "" {
		t.Fatalf("解锁失败: %v", err)
	}
	// 未锁定的券不能直接核销
	if _, err := svc.RedeemCoupon(coupon.ID, "order-1", 20000, ""); err == nil {
		t.Fatal("未锁定核销应被拒绝")
	}
	// 重新锁定并核销
	if _, err := svc.LockCoupon(coupon.ID, "order-1"); err != nil {
		t.Fatal(err)
	}
	record, err := svc.RedeemCoupon(coupon.ID, "order-1", 20000, "")
	if err != nil {
		t.Fatalf("核销失败: %v", err)
	}
	if record.DiscountAmt != 500 {
		t.Fatalf("满减券抵扣应为 500 分，得到 %d", record.DiscountAmt)
	}
	got, _ := svc.GetUserCoupon(coupon.ID)
	if got.Status != model.UserCouponUsed {
		t.Fatalf("核销后状态应为 used，得到 %s", got.Status)
	}
}

func TestRedeemBelowMinSpend(t *testing.T) {
	svc := newTestService()
	tpl := activeFixedTemplate(t, svc, "MINSPEND") // MinSpend=10000
	coupon := claimAndLock(t, svc, tpl.ID, "u1", "order-2")
	if _, err := svc.RedeemCoupon(coupon.ID, "order-2", 5000, ""); err == nil {
		t.Fatal("未达门槛核销应被拒绝")
	}
}

func TestRedeemWrongOrder(t *testing.T) {
	svc := newTestService()
	tpl := activeFixedTemplate(t, svc, "WRONGORDER")
	coupon := claimAndLock(t, svc, tpl.ID, "u1", "order-A")
	if _, err := svc.RedeemCoupon(coupon.ID, "order-B", 20000, ""); err == nil {
		t.Fatal("锁定订单与核销订单不一致应被拒绝")
	}
}

func TestCalcDiscount(t *testing.T) {
	if got := CalcDiscount(model.TypeFixed, 500, 20000); got != 500 {
		t.Fatalf("满减券抵扣期望 500，得到 %d", got)
	}
	// 8 折券：20000 * (100-80)/100 = 4000
	if got := CalcDiscount(model.TypePercent, 80, 20000); got != 4000 {
		t.Fatalf("折扣券抵扣期望 4000，得到 %d", got)
	}
}

func TestPercentCouponWithMaxDiscount(t *testing.T) {
	svc := newTestService()
	tpl, err := svc.CreateTemplate(model.CouponTemplate{
		Code: "PCT", Name: "8折封顶20元", Type: model.TypePercent,
		DiscountValue: 80, MaxDiscount: 2000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.TransitionTemplate(tpl.ID, model.TemplateActive); err != nil {
		t.Fatal(err)
	}
	coupon := claimAndLock(t, svc, tpl.ID, "u1", "order-pct")
	// 订单 50000 分，8 折抵扣 10000，封顶 2000
	record, err := svc.RedeemCoupon(coupon.ID, "order-pct", 50000, "")
	if err != nil {
		t.Fatal(err)
	}
	if record.DiscountAmt != 2000 {
		t.Fatalf("期望封顶抵扣 2000，得到 %d", record.DiscountAmt)
	}
}

func TestOrderCapRule(t *testing.T) {
	svc := newTestService()
	tpl := activeFixedTemplate(t, svc, "CAPRULE")
	if _, err := svc.CreateRule(model.Rule{
		Name: "单笔抵扣上限 3 元", Type: model.RuleOrderCap, TemplateID: tpl.ID, LimitValue: 300,
	}); err != nil {
		t.Fatal(err)
	}
	coupon := claimAndLock(t, svc, tpl.ID, "u1", "order-cap")
	record, err := svc.RedeemCoupon(coupon.ID, "order-cap", 20000, "")
	if err != nil {
		t.Fatal(err)
	}
	if record.DiscountAmt != 300 {
		t.Fatalf("期望被规则限制为 300，得到 %d", record.DiscountAmt)
	}
}

func TestCategoryExcludeRule(t *testing.T) {
	svc := newTestService()
	tpl := activeFixedTemplate(t, svc, "CATRULE")
	if _, err := svc.CreateRule(model.Rule{
		Name: "排除虚拟品类", Type: model.RuleCategoryExclude, TemplateID: tpl.ID, TextValue: "virtual",
	}); err != nil {
		t.Fatal(err)
	}
	coupon := claimAndLock(t, svc, tpl.ID, "u1", "order-cat")
	if _, err := svc.RedeemCoupon(coupon.ID, "order-cat", 20000, "virtual"); err == nil {
		t.Fatal("排除品类核销应被拒绝")
	}
	if _, err := svc.RedeemCoupon(coupon.ID, "order-cat", 20000, "food"); err != nil {
		t.Fatalf("正常品类核销失败: %v", err)
	}
}

func TestDailyUserLimitRule(t *testing.T) {
	svc := newTestService()
	tpl := activeFixedTemplate(t, svc, "DAILYRULE")
	if _, err := svc.CreateRule(model.Rule{
		Name: "每日限核销 1 次", Type: model.RuleDailyUserLimit, TemplateID: tpl.ID, LimitValue: 1,
	}); err != nil {
		t.Fatal(err)
	}
	c1 := claimAndLock(t, svc, tpl.ID, "u1", "order-d1")
	if _, err := svc.RedeemCoupon(c1.ID, "order-d1", 20000, ""); err != nil {
		t.Fatal(err)
	}
	c2 := claimAndLock(t, svc, tpl.ID, "u1", "order-d2")
	if _, err := svc.RedeemCoupon(c2.ID, "order-d2", 20000, ""); err == nil {
		t.Fatal("超过每日核销上限应被拒绝")
	}
}

func TestExpireCoupons(t *testing.T) {
	svc := newTestService()
	tpl := activeFixedTemplate(t, svc, "EXPIRE")
	coupon, err := svc.ClaimCoupon(tpl.ID, "u1")
	if err != nil {
		t.Fatal(err)
	}
	// 把过期时间改到过去
	coupon.ExpireAt = time.Now().Add(-time.Hour)
	if err := svc.store.UpdateUserCoupon(coupon); err != nil {
		t.Fatal(err)
	}
	if got := svc.ExpireCoupons(time.Now()); got != 1 {
		t.Fatalf("期望过期 1 张，得到 %d", got)
	}
	got, _ := svc.GetUserCoupon(coupon.ID)
	if got.Status != model.UserCouponExpired {
		t.Fatalf("期望 expired，得到 %s", got.Status)
	}
	// 已过期的券不能再锁定
	if _, err := svc.LockCoupon(coupon.ID, "order-x"); err == nil {
		t.Fatal("过期券锁定应被拒绝")
	}
}

func TestStatsOverviewAndGrouping(t *testing.T) {
	svc := newTestService()
	tpl := activeFixedTemplate(t, svc, "STATS")
	c1 := claimAndLock(t, svc, tpl.ID, "u1", "o1")
	if _, err := svc.RedeemCoupon(c1.ID, "o1", 20000, ""); err != nil {
		t.Fatal(err)
	}
	c2 := claimAndLock(t, svc, tpl.ID, "u2", "o2")
	if _, err := svc.RedeemCoupon(c2.ID, "o2", 30000, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ClaimCoupon(tpl.ID, "u3"); err != nil {
		t.Fatal(err)
	}

	overview := svc.Stats()
	if overview.IssuedCount != 3 || overview.UsedCount != 2 {
		t.Fatalf("概览数量不对: issued=%d used=%d", overview.IssuedCount, overview.UsedCount)
	}
	if overview.TotalDiscount != 1000 {
		t.Fatalf("累计抵扣期望 1000，得到 %d", overview.TotalDiscount)
	}
	if overview.TotalOrderAmt != 50000 {
		t.Fatalf("订单总额期望 50000，得到 %d", overview.TotalOrderAmt)
	}
	if overview.RedemptionRate != 67 {
		t.Fatalf("核销率期望 67%%，得到 %d", overview.RedemptionRate)
	}

	byTpl := svc.StatsByTemplate()
	if len(byTpl) != 1 || byTpl[0].UsedCount != 2 || byTpl[0].DiscountSum != 1000 {
		t.Fatalf("按模板统计不对: %+v", byTpl)
	}
	byDay := svc.StatsByDay()
	if len(byDay) != 1 || byDay[0].UsedCount != 2 {
		t.Fatalf("按天统计不对: %+v", byDay)
	}
	top := svc.TopUsers(10)
	if len(top) != 2 {
		t.Fatalf("期望 2 个用户，得到 %d", len(top))
	}
	if top[0].DiscountSum != 500 || top[1].DiscountSum != 500 {
		t.Fatalf("用户排行金额不对: %+v", top)
	}
}

func TestRuleCRUDAndValidation(t *testing.T) {
	svc := newTestService()
	tpl := activeFixedTemplate(t, svc, "RULECRUD")
	rule, err := svc.CreateRule(model.Rule{
		Name: "每日限 2 次", Type: model.RuleDailyUserLimit, TemplateID: tpl.ID, LimitValue: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateRule(model.Rule{
		Name: "挂错模板", Type: model.RuleOrderCap, TemplateID: "missing", LimitValue: 1,
	}); err == nil {
		t.Fatal("关联不存在模板应被拒绝")
	}
	if _, err := svc.CreateRule(model.Rule{Name: "", Type: model.RuleOrderCap, LimitValue: 1}); err == nil {
		t.Fatal("空名称应被拒绝")
	}
	updated, err := svc.UpdateRule(rule.ID, "", model.RuleInactive, 5, "")
	if err != nil || updated.Status != model.RuleInactive || updated.LimitValue != 5 {
		t.Fatalf("更新规则失败: %v %+v", err, updated)
	}
	items, total, err := svc.ListRules(model.RuleFilter{Type: model.RuleDailyUserLimit}, 1, 10)
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("规则筛选失败: total=%d err=%v", total, err)
	}
	if err := svc.DeleteRule(rule.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteRule(rule.ID); err == nil {
		t.Fatal("重复删除应报错")
	}
}

func TestRedeemCodeFlow(t *testing.T) {
	svc := newTestService()
	tpl := activeFixedTemplate(t, svc, "RCFLOW")
	rc, err := svc.CreateRedeemCode(tpl.ID, "WELCOME2026", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	// 重复码冲突
	if _, err := svc.CreateRedeemCode(tpl.ID, "WELCOME2026", time.Time{}); err == nil {
		t.Fatal("重复兑换码应冲突")
	}
	// 兑换
	coupon, err := svc.RedeemByCode("WELCOME2026", "u1")
	if err != nil {
		t.Fatalf("兑换失败: %v", err)
	}
	if coupon.TemplateID != tpl.ID {
		t.Fatal("兑换得到的券模板不对")
	}
	// 二次兑换应失败
	if _, err := svc.RedeemByCode("WELCOME2026", "u2"); err == nil {
		t.Fatal("已兑换的码应被拒绝")
	}
	got, _ := svc.GetRedeemCode(rc.ID)
	if got.Status != model.RedeemCodeRedeemed || got.RedeemedBy != "u1" {
		t.Fatalf("兑换码状态不对: %+v", got)
	}
	// 已兑换的码不能删除
	if err := svc.DeleteRedeemCode(rc.ID); err == nil {
		t.Fatal("已兑换的码删除应被拒绝")
	}
}

func TestRedeemCodeVoidAndExpire(t *testing.T) {
	svc := newTestService()
	tpl := activeFixedTemplate(t, svc, "RCVOID")
	rc, err := svc.CreateRedeemCode(tpl.ID, "VOIDME", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.VoidRedeemCode(rc.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RedeemByCode("VOIDME", "u1"); err == nil {
		t.Fatal("已作废的码兑换应被拒绝")
	}
	// 重复作废应失败
	if _, err := svc.VoidRedeemCode(rc.ID); err == nil {
		t.Fatal("重复作废应被拒绝")
	}
	// 过期码
	rc2, err := svc.CreateRedeemCode(tpl.ID, "EXPIRED1", time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RedeemByCode("EXPIRED1", "u1"); err == nil {
		t.Fatal("过期码兑换应被拒绝")
	}
	_ = rc2
}

func TestBatchCreateRedeemCodes(t *testing.T) {
	svc := newTestService()
	tpl := activeFixedTemplate(t, svc, "RCBATCH")
	count, err := svc.BatchCreateRedeemCodes(tpl.ID, 20, time.Time{})
	if err != nil || count != 20 {
		t.Fatalf("批量创建期望 20，得到 %d err=%v", count, err)
	}
	if _, err := svc.BatchCreateRedeemCodes(tpl.ID, 0, time.Time{}); err == nil {
		t.Fatal("数量为 0 应报错")
	}
	if _, err := svc.BatchCreateRedeemCodes("missing", 5, time.Time{}); err == nil {
		t.Fatal("模板不存在应报错")
	}
	items, total, err := svc.ListRedeemCodes(model.RedeemCodeFilter{TemplateID: tpl.ID}, 1, 100)
	if err != nil || total != 20 || len(items) != 20 {
		t.Fatalf("列表期望 20 条，得到 total=%d len=%d", total, len(items))
	}
}

func TestUsageRecordListFilter(t *testing.T) {
	svc := newTestService()
	tpl := activeFixedTemplate(t, svc, "USG")
	c1 := claimAndLock(t, svc, tpl.ID, "u1", "uo1")
	if _, err := svc.RedeemCoupon(c1.ID, "uo1", 20000, ""); err != nil {
		t.Fatal(err)
	}
	items, total, err := svc.ListUsageRecords(model.UsageFilter{UserID: "u1"}, 1, 10)
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("按用户筛选失败: total=%d err=%v", total, err)
	}
	if _, err := svc.GetUsageRecord(items[0].ID); err != nil {
		t.Fatal(err)
	}
	_, total, err = svc.ListUsageRecords(model.UsageFilter{OrderID: "uo1"}, 1, 10)
	if err != nil || total != 1 {
		t.Fatalf("按订单筛选失败: total=%d err=%v", total, err)
	}
	_, total, err = svc.ListUsageRecords(model.UsageFilter{UserID: "nobody"}, 1, 10)
	if err != nil || total != 0 {
		t.Fatalf("无匹配应返回 0，得到 %d", total)
	}
}
