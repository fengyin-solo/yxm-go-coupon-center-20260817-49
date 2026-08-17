package store

import (
	"couponcenter/internal/model"
)

func (s *MemoryStore) CreateUsageRecord(u *model.UsageRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, exist := range s.usageRecords {
		if exist.CouponID == u.CouponID {
			return ErrConflict
		}
	}
	s.usageRecords[u.ID] = u
	return nil
}

func (s *MemoryStore) GetUsageRecord(id string) (*model.UsageRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.usageRecords[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (s *MemoryStore) ListUsageRecords() []*model.UsageRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*model.UsageRecord, 0, len(s.usageRecords))
	for _, u := range s.usageRecords {
		cp := *u
		list = append(list, &cp)
	}
	return list
}
