package model

import (
	"strings"
	"time"
)

// 兑换码状态。
const (
	RedeemCodeAvailable = "available" // 待兑换
	RedeemCodeRedeemed  = "redeemed"  // 已兑换，终态
	RedeemCodeVoid      = "void"      // 已作废，终态
)

// redeemCodeTransitions 兑换码状态机：available -> redeemed / void。
var redeemCodeTransitions = map[string]map[string]bool{
	RedeemCodeAvailable: {RedeemCodeRedeemed: true, RedeemCodeVoid: true},
}

// CanRedeemCodeTransition 判断兑换码状态流转是否合法。
func CanRedeemCodeTransition(from, to string) bool {
	if m, ok := redeemCodeTransitions[from]; ok {
		return m[to]
	}
	return false
}

// RedemptionCode 兑换码，用户输入码即可兑换对应模板的一张优惠券。
type RedemptionCode struct {
	ID         string    `json:"id"`
	Code       string    `json:"code"`        // 兑换码，全局唯一
	TemplateID string    `json:"template_id"` // 兑换后领取的模板
	Status     string    `json:"status"`
	RedeemedBy string    `json:"redeemed_by"` // 兑换用户
	RedeemedAt time.Time `json:"redeemed_at"`
	ExpireAt   time.Time `json:"expire_at"` // 兑换码自身的截止时间
	CreatedAt  time.Time `json:"created_at"`
}

// Validate 校验兑换码字段。
func (c *RedemptionCode) Validate() error {
	c.Code = strings.TrimSpace(c.Code)
	c.TemplateID = strings.TrimSpace(c.TemplateID)
	c.RedeemedBy = strings.TrimSpace(c.RedeemedBy)
	if c.Code == "" {
		return NewValidationError("code", "兑换码不能为空")
	}
	if len(c.Code) > 24 {
		return NewValidationError("code", "兑换码不能超过 24 个字符")
	}
	if c.TemplateID == "" {
		return NewValidationError("template_id", "模板 ID 不能为空")
	}
	if c.Status == "" {
		c.Status = RedeemCodeAvailable
	}
	switch c.Status {
	case RedeemCodeAvailable, RedeemCodeRedeemed, RedeemCodeVoid:
	default:
		return NewValidationError("status", "兑换码状态不合法")
	}
	return nil
}

// IsExpired 判断兑换码在给定时间点是否已过期（仅对未兑换的码有意义）。
func (c *RedemptionCode) IsExpired(now time.Time) bool {
	if c.Status != RedeemCodeAvailable {
		return false
	}
	return !c.ExpireAt.IsZero() && now.After(c.ExpireAt)
}

// RedeemCodeFilter 兑换码列表筛选条件。
type RedeemCodeFilter struct {
	TemplateID string
	Status     string
	Keyword    string
}

// Match 判断兑换码是否满足筛选条件。
func (f RedeemCodeFilter) Match(c *RedemptionCode) bool {
	if f.TemplateID != "" && c.TemplateID != f.TemplateID {
		return false
	}
	if f.Status != "" && c.Status != f.Status {
		return false
	}
	if f.Keyword != "" {
		k := strings.ToLower(strings.TrimSpace(f.Keyword))
		if k != "" && !strings.Contains(strings.ToLower(c.Code), k) {
			return false
		}
	}
	return true
}
