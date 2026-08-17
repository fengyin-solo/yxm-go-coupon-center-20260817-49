package model

import (
	"strings"
	"time"
)

// UsageRecord 优惠券核销使用记录，核销后不可修改，仅支持查询与统计。
// 金额字段单位均为人民币「分」。
type UsageRecord struct {
	ID          string    `json:"id"`
	CouponID    string    `json:"coupon_id"`    // 用户券 ID
	TemplateID  string    `json:"template_id"`  // 模板 ID
	UserID      string    `json:"user_id"`      // 用户 ID
	OrderID     string    `json:"order_id"`     // 订单号
	OrderAmount int64     `json:"order_amount"` // 订单金额（分）
	DiscountAmt int64     `json:"discount_amt"` // 实际抵扣金额（分）
	UsedAt      time.Time `json:"used_at"`      // 核销时间
}

// Validate 校验使用记录字段。
func (u *UsageRecord) Validate() error {
	u.CouponID = strings.TrimSpace(u.CouponID)
	u.TemplateID = strings.TrimSpace(u.TemplateID)
	u.UserID = strings.TrimSpace(u.UserID)
	u.OrderID = strings.TrimSpace(u.OrderID)
	if u.CouponID == "" {
		return NewValidationError("coupon_id", "用户券 ID 不能为空")
	}
	if u.TemplateID == "" {
		return NewValidationError("template_id", "模板 ID 不能为空")
	}
	if u.UserID == "" {
		return NewValidationError("user_id", "用户 ID 不能为空")
	}
	if u.OrderID == "" {
		return NewValidationError("order_id", "订单号不能为空")
	}
	if u.OrderAmount <= 0 {
		return NewValidationError("order_amount", "订单金额必须大于 0")
	}
	if u.DiscountAmt <= 0 {
		return NewValidationError("discount_amt", "抵扣金额必须大于 0")
	}
	if u.DiscountAmt > u.OrderAmount {
		return NewValidationError("discount_amt", "抵扣金额不能超过订单金额")
	}
	if u.UsedAt.IsZero() {
		return NewValidationError("used_at", "核销时间不能为空")
	}
	return nil
}

// UsageFilter 使用记录筛选条件。
type UsageFilter struct {
	UserID     string
	TemplateID string
	OrderID    string
	From       time.Time
	To         time.Time
}

// Match 判断使用记录是否满足筛选条件。
func (f UsageFilter) Match(u *UsageRecord) bool {
	if f.UserID != "" && u.UserID != f.UserID {
		return false
	}
	if f.TemplateID != "" && u.TemplateID != f.TemplateID {
		return false
	}
	if f.OrderID != "" && u.OrderID != f.OrderID {
		return false
	}
	if !f.From.IsZero() && u.UsedAt.Before(f.From) {
		return false
	}
	if !f.To.IsZero() && u.UsedAt.After(f.To) {
		return false
	}
	return true
}
