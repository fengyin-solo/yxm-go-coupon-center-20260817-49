package service

import (
	"sort"
	"time"

	"couponcenter/internal/model"
	"couponcenter/pkg/idgen"
)

// CreateRule 创建使用规则；若指定了 TemplateID 则校验模板存在。
func (s *Service) CreateRule(input model.Rule) (*model.Rule, error) {
	input.ID = ""
	if err := input.Validate(); err != nil {
		return nil, err
	}
	if input.TemplateID != "" {
		if _, err := s.store.GetTemplate(input.TemplateID); err != nil {
			return nil, model.NewValidationError("template_id", "关联模板不存在")
		}
	}
	now := time.Now()
	input.ID = idgen.Hex()
	input.CreatedAt = now
	input.UpdatedAt = now
	if err := s.store.CreateRule(&input); err != nil {
		return nil, err
	}
	s.log.Infof("创建规则 %s(%s)", input.Name, input.Type)
	return &input, nil
}

// GetRule 查询规则详情。
func (s *Service) GetRule(id string) (*model.Rule, error) {
	return s.store.GetRule(id)
}

// ListRules 分页查询规则列表。
func (s *Service) ListRules(filter model.RuleFilter, page, size int) ([]*model.Rule, int, error) {
	if err := normalizePage(page, size); err != nil {
		return nil, 0, err
	}
	all := s.store.ListRules()
	matched := make([]*model.Rule, 0, len(all))
	for _, r := range all {
		if filter.Match(r) {
			matched = append(matched, r)
		}
	}
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].CreatedAt.After(matched[j].CreatedAt)
	})
	total := len(matched)
	start := (page - 1) * size
	if start >= total {
		return []*model.Rule{}, total, nil
	}
	end := start + size
	if end > total {
		end = total
	}
	return matched[start:end], total, nil
}

// UpdateRule 更新规则字段。
func (s *Service) UpdateRule(id, name, status string, limitValue int64, textValue string) (*model.Rule, error) {
	r, err := s.store.GetRule(id)
	if err != nil {
		return nil, err
	}
	if name != "" {
		r.Name = name
	}
	if status != "" {
		r.Status = status
	}
	if limitValue > 0 {
		r.LimitValue = limitValue
	}
	if textValue != "" {
		r.TextValue = textValue
	}
	r.UpdatedAt = time.Now()
	if err := r.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.UpdateRule(r); err != nil {
		return nil, err
	}
	return r, nil
}

// DeleteRule 删除规则。
func (s *Service) DeleteRule(id string) error {
	if _, err := s.store.GetRule(id); err != nil {
		return err
	}
	return s.store.DeleteRule(id)
}
