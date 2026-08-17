// Package store 定义数据访问接口与内存实现。
package store

import (
	"errors"

	"couponcenter/internal/model"
)

var (
	ErrNotFound = errors.New("记录不存在")
	ErrConflict = errors.New("记录已存在或状态冲突")
)

// Store 聚合全部实体的数据访问方法，便于测试时替换实现。
type Store interface {
	// 优惠券模板
	CreateTemplate(t *model.CouponTemplate) error
	GetTemplate(id string) (*model.CouponTemplate, error)
	GetTemplateByCode(code string) (*model.CouponTemplate, error)
	ListTemplates() []*model.CouponTemplate
	UpdateTemplate(t *model.CouponTemplate) error
	DeleteTemplate(id string) error

	// 用户券
	CreateUserCoupon(c *model.UserCoupon) error
	GetUserCoupon(id string) (*model.UserCoupon, error)
	ListUserCoupons() []*model.UserCoupon
	UpdateUserCoupon(c *model.UserCoupon) error
	DeleteUserCoupon(id string) error
	CountUserCouponsByTemplate(userID, templateID string) int

	// 使用记录
	CreateUsageRecord(u *model.UsageRecord) error
	GetUsageRecord(id string) (*model.UsageRecord, error)
	ListUsageRecords() []*model.UsageRecord

	// 规则
	CreateRule(r *model.Rule) error
	GetRule(id string) (*model.Rule, error)
	ListRules() []*model.Rule
	UpdateRule(r *model.Rule) error
	DeleteRule(id string) error

	// 兑换码
	CreateRedeemCode(c *model.RedemptionCode) error
	GetRedeemCode(id string) (*model.RedemptionCode, error)
	GetRedeemCodeByCode(code string) (*model.RedemptionCode, error)
	ListRedeemCodes() []*model.RedemptionCode
	UpdateRedeemCode(c *model.RedemptionCode) error
	DeleteRedeemCode(id string) error
}
