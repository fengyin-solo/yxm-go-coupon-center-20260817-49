package model

import (
	"strings"
	"time"
)

// 优惠券模板状态。
const (
	TemplateDraft    = "draft"    // 草稿，不可领取
	TemplateActive   = "active"   // 生效中，可领取
	TemplatePaused   = "paused"   // 暂停发放
	TemplateArchived = "archived" // 已归档，终态
)

// 优惠券类型。
const (
	TypeFixed   = "fixed"   // 满减券：DiscountValue 为抵扣金额（单位：分）
	TypePercent = "percent" // 折扣券：DiscountValue 为折扣百分比（1-99），抵扣 = 订单金额 * (100-DiscountValue) / 100
)

// templateTransitions 模板状态机：draft -> active <-> paused，active/paused -> archived。
var templateTransitions = map[string]map[string]bool{
	TemplateDraft:  {TemplateActive: true},
	TemplateActive: {TemplatePaused: true, TemplateArchived: true},
	TemplatePaused: {TemplateActive: true, TemplateArchived: true},
}

// CanTemplateTransition 判断模板状态流转是否合法。
func CanTemplateTransition(from, to string) bool {
	if m, ok := templateTransitions[from]; ok {
		return m[to]
	}
	return false
}

// CouponTemplate 优惠券模板（批次），定义一类优惠券的面额、库存与有效期。
// 所有金额字段单位均为人民币「分」。
type CouponTemplate struct {
	ID            string    `json:"id"`
	Code          string    `json:"code"`           // 模板编码，全局唯一
	Name          string    `json:"name"`           // 模板名称
	Type          string    `json:"type"`           // fixed / percent
	DiscountValue int64     `json:"discount_value"` // fixed: 抵扣分；percent: 折扣百分比
	MinSpend      int64     `json:"min_spend"`      // 使用门槛（分），0 表示无门槛
	MaxDiscount   int64     `json:"max_discount"`   // percent 类型的抵扣上限（分），0 表示不限
	TotalQuantity int       `json:"total_quantity"` // 发行总量，0 表示不限量
	IssuedCount   int       `json:"issued_count"`   // 已领取数量
	PerUserLimit  int       `json:"per_user_limit"` // 每人限领，0 表示不限
	ValidDays     int       `json:"valid_days"`     // 领取后有效天数，0 表示以 EndTime 为准
	Category      string    `json:"category"`       // 适用品类标签
	StartTime     time.Time `json:"start_time"`     // 发放开始时间
	EndTime       time.Time `json:"end_time"`       // 发放截止时间
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Validate 校验并规范化模板字段。
func (t *CouponTemplate) Validate() error {
	t.Code = strings.TrimSpace(t.Code)
	t.Name = strings.TrimSpace(t.Name)
	t.Category = strings.TrimSpace(t.Category)
	if t.Code == "" {
		return NewValidationError("code", "模板编码不能为空")
	}
	if len(t.Code) > 32 {
		return NewValidationError("code", "模板编码不能超过 32 个字符")
	}
	if t.Name == "" {
		return NewValidationError("name", "模板名称不能为空")
	}
	switch t.Type {
	case TypeFixed:
		if t.DiscountValue <= 0 {
			return NewValidationError("discount_value", "满减券抵扣金额必须大于 0")
		}
	case TypePercent:
		if t.DiscountValue < 1 || t.DiscountValue > 99 {
			return NewValidationError("discount_value", "折扣券百分比必须在 1-99 之间")
		}
		if t.MaxDiscount < 0 {
			return NewValidationError("max_discount", "抵扣上限不能为负数")
		}
	default:
		return NewValidationError("type", "优惠券类型不合法，仅支持 fixed / percent")
	}
	if t.MinSpend < 0 {
		return NewValidationError("min_spend", "使用门槛不能为负数")
	}
	if t.TotalQuantity < 0 {
		return NewValidationError("total_quantity", "发行总量不能为负数")
	}
	if t.PerUserLimit < 0 {
		return NewValidationError("per_user_limit", "每人限领数量不能为负数")
	}
	if t.ValidDays < 0 {
		return NewValidationError("valid_days", "有效天数不能为负数")
	}
	if !t.EndTime.IsZero() && !t.StartTime.IsZero() && t.EndTime.Before(t.StartTime) {
		return NewValidationError("end_time", "发放截止时间不能早于开始时间")
	}
	if t.Status == "" {
		t.Status = TemplateDraft
	}
	switch t.Status {
	case TemplateDraft, TemplateActive, TemplatePaused, TemplateArchived:
	default:
		return NewValidationError("status", "模板状态不合法")
	}
	return nil
}

// Remaining 剩余可发放数量，-1 表示不限量。
func (t *CouponTemplate) Remaining() int {
	if t.TotalQuantity == 0 {
		return -1
	}
	return t.TotalQuantity - t.IssuedCount
}

// TemplateFilter 模板列表筛选条件。
type TemplateFilter struct {
	Status   string
	Type     string
	Category string
	Keyword  string
}

// Match 判断模板是否满足筛选条件。
func (f TemplateFilter) Match(t *CouponTemplate) bool {
	if f.Status != "" && t.Status != f.Status {
		return false
	}
	if f.Type != "" && t.Type != f.Type {
		return false
	}
	if f.Category != "" && t.Category != f.Category {
		return false
	}
	if f.Keyword != "" {
		k := strings.ToLower(strings.TrimSpace(f.Keyword))
		if k != "" && !strings.Contains(strings.ToLower(t.Name), k) &&
			!strings.Contains(strings.ToLower(t.Code), k) {
			return false
		}
	}
	return true
}
