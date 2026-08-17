package service

import (
	"sort"
	"time"

	"couponcenter/internal/model"
	"couponcenter/internal/store"
	"couponcenter/pkg/idgen"
)

// CreateTemplate 创建优惠券模板（初始为草稿态）。
func (s *Service) CreateTemplate(input model.CouponTemplate) (*model.CouponTemplate, error) {
	input.ID = ""
	input.IssuedCount = 0
	input.Status = model.TemplateDraft
	if err := input.Validate(); err != nil {
		return nil, err
	}
	now := time.Now()
	input.ID = idgen.Hex()
	input.CreatedAt = now
	input.UpdatedAt = now
	if err := s.store.CreateTemplate(&input); err != nil {
		return nil, err
	}
	s.log.Infof("创建优惠券模板 %s(%s)", input.Name, input.Code)
	return &input, nil
}

// GetTemplate 查询模板详情。
func (s *Service) GetTemplate(id string) (*model.CouponTemplate, error) {
	return s.store.GetTemplate(id)
}

// ListTemplates 分页查询模板列表。
func (s *Service) ListTemplates(filter model.TemplateFilter, page, size int) ([]*model.CouponTemplate, int, error) {
	if err := normalizePage(page, size); err != nil {
		return nil, 0, err
	}
	all := s.store.ListTemplates()
	matched := make([]*model.CouponTemplate, 0, len(all))
	for _, t := range all {
		if filter.Match(t) {
			matched = append(matched, t)
		}
	}
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].CreatedAt.After(matched[j].CreatedAt)
	})
	total := len(matched)
	start := (page - 1) * size
	if start >= total {
		return []*model.CouponTemplate{}, total, nil
	}
	end := start + size
	if end > total {
		end = total
	}
	return matched[start:end], total, nil
}

// UpdateTemplate 更新模板可编辑字段（不含发行量与状态）。
func (s *Service) UpdateTemplate(id string, name, category string, minSpend, maxDiscount int64, perUserLimit int) (*model.CouponTemplate, error) {
	t, err := s.store.GetTemplate(id)
	if err != nil {
		return nil, err
	}
	if t.Status == model.TemplateArchived {
		return nil, model.NewValidationError("status", "已归档模板不可编辑")
	}
	if name != "" {
		t.Name = name
	}
	if category != "" {
		t.Category = category
	}
	if minSpend >= 0 {
		t.MinSpend = minSpend
	}
	if maxDiscount >= 0 {
		t.MaxDiscount = maxDiscount
	}
	if perUserLimit >= 0 {
		t.PerUserLimit = perUserLimit
	}
	t.UpdatedAt = time.Now()
	if err := t.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.UpdateTemplate(t); err != nil {
		return nil, err
	}
	return t, nil
}

// TransitionTemplate 模板状态流转（draft->active<->paused，active/paused->archived）。
func (s *Service) TransitionTemplate(id, to string) (*model.CouponTemplate, error) {
	t, err := s.store.GetTemplate(id)
	if err != nil {
		return nil, err
	}
	if !model.CanTemplateTransition(t.Status, to) {
		return nil, model.NewValidationError("status", "模板状态不允许从 "+t.Status+" 流转到 "+to)
	}
	t.Status = to
	t.UpdatedAt = time.Now()
	if err := s.store.UpdateTemplate(t); err != nil {
		return nil, err
	}
	s.log.Infof("模板 %s 状态流转为 %s", t.Code, to)
	return t, nil
}

// DeleteTemplate 删除模板；若已有用户领取则拒绝删除。
func (s *Service) DeleteTemplate(id string) error {
	if _, err := s.store.GetTemplate(id); err != nil {
		return err
	}
	for _, c := range s.store.ListUserCoupons() {
		if c.TemplateID == id {
			return store.ErrConflict
		}
	}
	return s.store.DeleteTemplate(id)
}

// BatchArchiveTemplates 批量归档模板，返回成功归档的数量；
// 只有 active/paused 状态的模板会被归档，其余跳过。
func (s *Service) BatchArchiveTemplates(ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, model.NewValidationError("ids", "模板 ID 列表不能为空")
	}
	archived := 0
	for _, id := range ids {
		t, err := s.store.GetTemplate(id)
		if err != nil {
			continue
		}
		if !model.CanTemplateTransition(t.Status, model.TemplateArchived) {
			continue
		}
		t.Status = model.TemplateArchived
		t.UpdatedAt = time.Now()
		if err := s.store.UpdateTemplate(t); err != nil {
			continue
		}
		archived++
	}
	s.log.Infof("批量归档模板：请求 %d 个，成功 %d 个", len(ids), archived)
	return archived, nil
}
