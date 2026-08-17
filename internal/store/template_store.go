package store

import (
	"couponcenter/internal/model"
)

func (s *MemoryStore) CreateTemplate(t *model.CouponTemplate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, exist := range s.templates {
		if exist.Code == t.Code {
			return ErrConflict
		}
	}
	s.templates[t.ID] = t
	return nil
}

func (s *MemoryStore) GetTemplate(id string) (*model.CouponTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.templates[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *t
	return &cp, nil
}

func (s *MemoryStore) GetTemplateByCode(code string) (*model.CouponTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, t := range s.templates {
		if t.Code == code {
			cp := *t
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

func (s *MemoryStore) ListTemplates() []*model.CouponTemplate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*model.CouponTemplate, 0, len(s.templates))
	for _, t := range s.templates {
		cp := *t
		list = append(list, &cp)
	}
	return list
}

func (s *MemoryStore) UpdateTemplate(t *model.CouponTemplate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.templates[t.ID]; !ok {
		return ErrNotFound
	}
	for _, exist := range s.templates {
		if exist.ID != t.ID && exist.Code == t.Code {
			return ErrConflict
		}
	}
	s.templates[t.ID] = t
	return nil
}

func (s *MemoryStore) DeleteTemplate(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.templates[id]; !ok {
		return ErrNotFound
	}
	delete(s.templates, id)
	return nil
}
