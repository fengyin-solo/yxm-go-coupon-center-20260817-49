package model

import (
	"strings"
	"time"
)

// 用户券状态。
const (
	UserCouponUnused  = "unused"  // 未使用
	UserCouponLocked  = "locked"  // 已锁定（下单占用）
	UserCouponUsed    = "used"    // 已核销，终态
	UserCouponExpired = "expired" // 已过期，终态
)

// userCouponTransitions 用户券状态机：unused -> locked -> used，unused/locked -> expired。
var userCouponTransitions = map[string]map[string]bool{
	UserCouponUnused: {UserCouponLocked: true, UserCouponExpired: true},
	UserCouponLocked: {UserCouponUnused: true, UserCouponUsed: true, UserCouponExpired: true},
}

// CanUserCouponTransition 判断用户券状态流转是否合法。
func CanUserCouponTransition(from, to string) bool {
	if m, ok := userCouponTransitions[from]; ok {
		return m[to]
	}
	return false
}

// UserCoupon 用户领取的一张优惠券实例。
// 所有金额字段单位均为人民币「分」。
type UserCoupon struct {
	ID         string    `json:"id"`
	TemplateID string    `json:"template_id"`
	UserID     string    `json:"user_id"`
	Type       string    `json:"type"`      // 冗余自模板：fixed / percent
	Discount   int64     `json:"discount"`  // 冗余自模板：抵扣金额（分）或折扣百分比
	MinSpend   int64     `json:"min_spend"` // 冗余自模板：使用门槛（分）
	OrderID    string    `json:"order_id"`  // 锁定/核销时关联的订单号
	ExpireAt   time.Time `json:"expire_at"` // 过期时间
	Status     string    `json:"status"`
	ReceivedAt time.Time `json:"received_at"` // 领取时间
	UsedAt     time.Time `json:"used_at"`     // 核销时间
}

// Validate 校验用户券字段。
func (c *UserCoupon) Validate() error {
	c.TemplateID = strings.TrimSpace(c.TemplateID)
	c.UserID = strings.TrimSpace(c.UserID)
	c.OrderID = strings.TrimSpace(c.OrderID)
	if c.TemplateID == "" {
		return NewValidationError("template_id", "模板 ID 不能为空")
	}
	if c.UserID == "" {
		return NewValidationError("user_id", "用户 ID 不能为空")
	}
	if c.Type != TypeFixed && c.Type != TypePercent {
		return NewValidationError("type", "优惠券类型不合法")
	}
	if c.Discount <= 0 {
		return NewValidationError("discount", "优惠面值必须大于 0")
	}
	if c.MinSpend < 0 {
		return NewValidationError("min_spend", "使用门槛不能为负数")
	}
	if c.ExpireAt.IsZero() {
		return NewValidationError("expire_at", "过期时间不能为空")
	}
	if c.Status == "" {
		c.Status = UserCouponUnused
	}
	switch c.Status {
	case UserCouponUnused, UserCouponLocked, UserCouponUsed, UserCouponExpired:
	default:
		return NewValidationError("status", "用户券状态不合法")
	}
	return nil
}

// IsExpired 判断券在给定时间点是否已过期（未使用/未锁定的券过期后视为过期）。
func (c *UserCoupon) IsExpired(now time.Time) bool {
	if c.Status == UserCouponUsed || c.Status == UserCouponExpired {
		return false
	}
	return now.After(c.ExpireAt)
}

// UserCouponFilter 用户券列表筛选条件。
type UserCouponFilter struct {
	UserID     string
	TemplateID string
	Status     string
}

// Match 判断用户券是否满足筛选条件。
func (f UserCouponFilter) Match(c *UserCoupon) bool {
	if f.UserID != "" && c.UserID != f.UserID {
		return false
	}
	if f.TemplateID != "" && c.TemplateID != f.TemplateID {
		return false
	}
	if f.Status != "" && c.Status != f.Status {
		return false
	}
	return true
}
