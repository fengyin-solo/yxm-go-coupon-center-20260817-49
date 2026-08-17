package model

import (
	"strings"
	"time"
)

// 规则类型。
const (
	RuleDailyUserLimit  = "daily_user_limit" // 单用户每日核销上限，LimitValue 为次数
	RuleOrderCap        = "order_cap"        // 单笔订单抵扣上限（分），LimitValue 为金额
	RuleCategoryExclude = "category_exclude" // 排除品类，TextValue 为品类名
)

// 规则状态。
const (
	RuleActive   = "active"
	RuleInactive = "inactive"
)

// Rule 优惠券使用规则，可作用于全局（TemplateID 为空）或指定模板。
// 金额字段单位均为人民币「分」。
type Rule struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`        // daily_user_limit / order_cap / category_exclude
	TemplateID string    `json:"template_id"` // 为空表示全局规则
	LimitValue int64     `json:"limit_value"` // 数值型限制（次数或金额）
	TextValue  string    `json:"text_value"`  // 文本型限制（如品类名）
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Validate 校验规则字段。
func (r *Rule) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	r.TemplateID = strings.TrimSpace(r.TemplateID)
	r.TextValue = strings.TrimSpace(r.TextValue)
	if r.Name == "" {
		return NewValidationError("name", "规则名称不能为空")
	}
	switch r.Type {
	case RuleDailyUserLimit:
		if r.LimitValue <= 0 {
			return NewValidationError("limit_value", "每日核销上限必须大于 0")
		}
	case RuleOrderCap:
		if r.LimitValue <= 0 {
			return NewValidationError("limit_value", "订单抵扣上限必须大于 0")
		}
	case RuleCategoryExclude:
		if r.TextValue == "" {
			return NewValidationError("text_value", "排除品类不能为空")
		}
	default:
		return NewValidationError("type", "规则类型不合法")
	}
	if r.Status == "" {
		r.Status = RuleActive
	}
	if r.Status != RuleActive && r.Status != RuleInactive {
		return NewValidationError("status", "规则状态不合法")
	}
	return nil
}

// AppliesTo 判断规则是否作用于指定模板（全局规则作用于所有模板）。
func (r *Rule) AppliesTo(templateID string) bool {
	if r.Status != RuleActive {
		return false
	}
	// TemplateID 为空表示全局规则，对所有模板生效。
	if r.TemplateID == "" {
		return true
	}
	return r.TemplateID == templateID
}

// RuleFilter 规则列表筛选条件。
type RuleFilter struct {
	Type       string
	Status     string
	TemplateID string
}

// Match 判断规则是否满足筛选条件。
func (f RuleFilter) Match(r *Rule) bool {
	if f.Type != "" && r.Type != f.Type {
		return false
	}
	if f.Status != "" && r.Status != f.Status {
		return false
	}
	if f.TemplateID != "" && r.TemplateID != f.TemplateID {
		return false
	}
	return true
}
