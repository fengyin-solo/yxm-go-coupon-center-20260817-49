package store

import (
	"sync"

	"couponcenter/internal/model"
)

// MemoryStore 基于内存 map 的 Store 实现，所有操作线程安全。
type MemoryStore struct {
	mu           sync.RWMutex
	templates    map[string]*model.CouponTemplate
	userCoupons  map[string]*model.UserCoupon
	usageRecords map[string]*model.UsageRecord
	rules        map[string]*model.Rule
	redeemCodes  map[string]*model.RedemptionCode
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		templates:    make(map[string]*model.CouponTemplate),
		userCoupons:  make(map[string]*model.UserCoupon),
		usageRecords: make(map[string]*model.UsageRecord),
		rules:        make(map[string]*model.Rule),
		redeemCodes:  make(map[string]*model.RedemptionCode),
	}
}

var _ Store = (*MemoryStore)(nil)
