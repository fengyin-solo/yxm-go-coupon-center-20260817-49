package store

import (
	"couponcenter/internal/model"
)

func (s *MemoryStore) CreateRedeemCode(c *model.RedemptionCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, exist := range s.redeemCodes {
		if exist.Code == c.Code {
			return ErrConflict
		}
	}
	s.redeemCodes[c.ID] = c
	return nil
}

func (s *MemoryStore) GetRedeemCode(id string) (*model.RedemptionCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.redeemCodes[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *c
	return &cp, nil
}

func (s *MemoryStore) GetRedeemCodeByCode(code string) (*model.RedemptionCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.redeemCodes {
		if c.Code == code {
			cp := *c
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

func (s *MemoryStore) ListRedeemCodes() []*model.RedemptionCode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*model.RedemptionCode, 0, len(s.redeemCodes))
	for _, c := range s.redeemCodes {
		cp := *c
		list = append(list, &cp)
	}
	return list
}

func (s *MemoryStore) UpdateRedeemCode(c *model.RedemptionCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.redeemCodes[c.ID]; !ok {
		return ErrNotFound
	}
	for _, exist := range s.redeemCodes {
		if exist.ID != c.ID && exist.Code == c.Code {
			return ErrConflict
		}
	}
	s.redeemCodes[c.ID] = c
	return nil
}

func (s *MemoryStore) DeleteRedeemCode(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.redeemCodes[id]; !ok {
		return ErrNotFound
	}
	delete(s.redeemCodes, id)
	return nil
}
