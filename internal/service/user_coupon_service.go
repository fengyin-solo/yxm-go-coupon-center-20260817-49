package service

import (
	"sort"
	"time"

	"couponcenter/internal/model"
	"couponcenter/pkg/idgen"
)

// ClaimCoupon 用户领取优惠券。
// 校验链：模板存在 -> 模板处于 active -> 在发放时间窗内 -> 库存充足 -> 每人限领。
func (s *Service) ClaimCoupon(templateID, userID string) (*model.UserCoupon, error) {
	t, err := s.store.GetTemplate(templateID)
	if err != nil {
		return nil, err
	}
	if t.Status != model.TemplateActive {
		return nil, model.NewValidationError("template", "模板当前状态不可领取")
	}
	now := time.Now()
	if !t.StartTime.IsZero() && now.Before(t.StartTime) {
		return nil, model.NewValidationError("template", "发放尚未开始")
	}
	if !t.EndTime.IsZero() && now.After(t.EndTime) {
		return nil, model.NewValidationError("template", "发放已结束")
	}
	if t.Remaining() == 0 {
		return nil, model.NewValidationError("template", "优惠券已被领完")
	}
	if t.PerUserLimit > 0 {
		claimed := s.store.CountUserCouponsByTemplate(userID, templateID)
		if claimed >= t.PerUserLimit {
			return nil, model.NewValidationError("user", "已达到每人限领数量")
		}
	}

	expireAt := t.EndTime
	if t.ValidDays > 0 {
		byValidDays := now.AddDate(0, 0, t.ValidDays)
		if expireAt.IsZero() || byValidDays.Before(expireAt) {
			expireAt = byValidDays
		}
	}
	if expireAt.IsZero() {
		expireAt = now.AddDate(0, 3, 0) // 兜底：未配置有效期时默认 3 个月
	}

	coupon := &model.UserCoupon{
		ID:         idgen.Hex(),
		TemplateID: t.ID,
		UserID:     userID,
		Type:       t.Type,
		Discount:   t.DiscountValue,
		MinSpend:   t.MinSpend,
		ExpireAt:   expireAt,
		Status:     model.UserCouponUnused,
		ReceivedAt: now,
	}
	if err := coupon.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.CreateUserCoupon(coupon); err != nil {
		return nil, err
	}
	t.IssuedCount++
	t.UpdatedAt = now
	if err := s.store.UpdateTemplate(t); err != nil {
		return nil, err
	}
	s.log.Infof("用户 %s 领取优惠券 %s（模板 %s）", userID, coupon.ID, t.Code)
	return coupon, nil
}

// GetUserCoupon 查询用户券详情。
func (s *Service) GetUserCoupon(id string) (*model.UserCoupon, error) {
	return s.store.GetUserCoupon(id)
}

// ListUserCoupons 分页查询用户券列表。
func (s *Service) ListUserCoupons(filter model.UserCouponFilter, page, size int) ([]*model.UserCoupon, int, error) {
	if err := normalizePage(page, size); err != nil {
		return nil, 0, err
	}
	all := s.store.ListUserCoupons()
	matched := make([]*model.UserCoupon, 0, len(all))
	for _, c := range all {
		if filter.Match(c) {
			matched = append(matched, c)
		}
	}
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].ReceivedAt.After(matched[j].ReceivedAt)
	})
	total := len(matched)
	start := (page - 1) * size
	if start >= total {
		return []*model.UserCoupon{}, total, nil
	}
	end := start + size
	if end > total {
		end = total
	}
	return matched[start:end], total, nil
}

// LockCoupon 下单时锁定用户券。
func (s *Service) LockCoupon(id, orderID string) (*model.UserCoupon, error) {
	c, err := s.store.GetUserCoupon(id)
	if err != nil {
		return nil, err
	}
	if orderID == "" {
		return nil, model.NewValidationError("order_id", "订单号不能为空")
	}
	if c.IsExpired(time.Now()) {
		return nil, model.NewValidationError("coupon", "优惠券已过期")
	}
	if !model.CanUserCouponTransition(c.Status, model.UserCouponLocked) {
		return nil, model.NewValidationError("status", "当前状态不可锁定")
	}
	c.Status = model.UserCouponLocked
	c.OrderID = orderID
	if err := s.store.UpdateUserCoupon(c); err != nil {
		return nil, err
	}
	s.log.Infof("优惠券 %s 已锁定，关联订单 %s", id, orderID)
	return c, nil
}

// UnlockCoupon 取消订单时释放锁定的用户券，恢复为未使用。
func (s *Service) UnlockCoupon(id string) (*model.UserCoupon, error) {
	c, err := s.store.GetUserCoupon(id)
	if err != nil {
		return nil, err
	}
	if c.Status != model.UserCouponLocked {
		return nil, model.NewValidationError("status", "仅锁定状态的优惠券可解锁")
	}
	c.Status = model.UserCouponUnused
	c.OrderID = ""
	if err := s.store.UpdateUserCoupon(c); err != nil {
		return nil, err
	}
	s.log.Infof("优惠券 %s 已解锁", id)
	return c, nil
}

// RedeemCoupon 核销用户券：计算抵扣金额、校验规则、写入使用记录。
// orderAmount 为订单金额（分），category 为订单品类。
func (s *Service) RedeemCoupon(id, orderID string, orderAmount int64, category string) (*model.UsageRecord, error) {
	c, err := s.store.GetUserCoupon(id)
	if err != nil {
		return nil, err
	}
	if orderID == "" {
		return nil, model.NewValidationError("order_id", "订单号不能为空")
	}
	if orderAmount <= 0 {
		return nil, model.NewValidationError("order_amount", "订单金额必须大于 0")
	}
	if c.Status != model.UserCouponLocked || c.OrderID != orderID {
		return nil, model.NewValidationError("status", "优惠券未锁定到该订单，不可核销")
	}
	if time.Now().After(c.ExpireAt) {
		return nil, model.NewValidationError("coupon", "优惠券已过期")
	}
	if orderAmount < c.MinSpend {
		return nil, model.NewValidationError("order_amount", "订单金额未达到使用门槛")
	}

	// 品类排除规则校验
	t, err := s.store.GetTemplate(c.TemplateID)
	if err != nil {
		return nil, err
	}
	for _, r := range s.store.ListRules() {
		if r.Type == model.RuleCategoryExclude && r.AppliesTo(c.TemplateID) &&
			r.TextValue == category {
			return nil, model.NewValidationError("category", "该品类不可使用此优惠券")
		}
	}

	discountAmt := CalcDiscount(c.Type, c.Discount, orderAmount)
	if t.MaxDiscount > 0 && discountAmt > t.MaxDiscount {
		discountAmt = t.MaxDiscount
	}
	// 订单抵扣上限规则
	for _, r := range s.store.ListRules() {
		if r.Type == model.RuleOrderCap && r.AppliesTo(c.TemplateID) &&
			r.LimitValue > 0 && discountAmt > r.LimitValue {
			discountAmt = r.LimitValue
		}
	}
	if discountAmt > orderAmount {
		discountAmt = orderAmount
	}
	// 每日核销上限规则
	for _, r := range s.store.ListRules() {
		if r.Type == model.RuleDailyUserLimit && r.AppliesTo(c.TemplateID) {
			if s.countTodayUsage(c.UserID, time.Now()) >= int(r.LimitValue) {
				return nil, model.NewValidationError("user", "已达到今日核销上限")
			}
		}
	}

	now := time.Now()
	c.Status = model.UserCouponUsed
	c.UsedAt = now
	if err := s.store.UpdateUserCoupon(c); err != nil {
		return nil, err
	}
	record := &model.UsageRecord{
		ID:          idgen.Hex(),
		CouponID:    c.ID,
		TemplateID:  c.TemplateID,
		UserID:      c.UserID,
		OrderID:     orderID,
		OrderAmount: orderAmount,
		DiscountAmt: discountAmt,
		UsedAt:      now,
	}
	if err := record.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.CreateUsageRecord(record); err != nil {
		return nil, err
	}
	s.log.Infof("优惠券 %s 核销成功，订单 %s 抵扣 %d 分", id, orderID, discountAmt)
	return record, nil
}

// CalcDiscount 根据券类型计算抵扣金额（分）。
// fixed：直接返回面值；percent：订单金额 * (100-折扣) / 100。
func CalcDiscount(couponType string, discountValue, orderAmount int64) int64 {
	switch couponType {
	case model.TypePercent:
		return orderAmount * (100 - discountValue) / 100
	default:
		return discountValue
	}
}

// countTodayUsage 统计用户当天已核销次数。
func (s *Service) countTodayUsage(userID string, now time.Time) int {
	count := 0
	y, m, d := now.Date()
	dayStart := time.Date(y, m, d, 0, 0, 0, 0, now.Location())
	dayEnd := dayStart.AddDate(0, 0, 1)
	for _, u := range s.store.ListUsageRecords() {
		if u.UserID == userID && !u.UsedAt.Before(dayStart) && u.UsedAt.Before(dayEnd) {
			count++
		}
	}
	return count
}

// ExpireCoupons 扫描并将已过期的未使用/锁定券置为过期，返回处理数量。
func (s *Service) ExpireCoupons(now time.Time) int {
	expired := 0
	for _, c := range s.store.ListUserCoupons() {
		if c.IsExpired(now) {
			c.Status = model.UserCouponExpired
			if err := s.store.UpdateUserCoupon(c); err == nil {
				expired++
			}
		}
	}
	if expired > 0 {
		s.log.Infof("过期扫描：共 %d 张券置为过期", expired)
	}
	return expired
}
