package store

import (
	"couponcenter/internal/model"
)

func (s *MemoryStore) CreateUserCoupon(c *model.UserCoupon) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userCoupons[c.ID] = c
	return nil
}

func (s *MemoryStore) GetUserCoupon(id string) (*model.UserCoupon, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.userCoupons[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *c
	return &cp, nil
}

func (s *MemoryStore) ListUserCoupons() []*model.UserCoupon {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*model.UserCoupon, 0, len(s.userCoupons))
	for _, c := range s.userCoupons {
		cp := *c
		list = append(list, &cp)
	}
	return list
}

func (s *MemoryStore) UpdateUserCoupon(c *model.UserCoupon) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.userCoupons[c.ID]; !ok {
		return ErrNotFound
	}
	s.userCoupons[c.ID] = c
	return nil
}

func (s *MemoryStore) DeleteUserCoupon(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.userCoupons[id]; !ok {
		return ErrNotFound
	}
	delete(s.userCoupons, id)
	return nil
}

// CountUserCouponsByTemplate 统计某用户对某模板的领券数量（含所有状态）。
func (s *MemoryStore) CountUserCouponsByTemplate(userID, templateID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, c := range s.userCoupons {
		if c.UserID == userID && c.TemplateID == templateID {
			count++
		}
	}
	return count
}
