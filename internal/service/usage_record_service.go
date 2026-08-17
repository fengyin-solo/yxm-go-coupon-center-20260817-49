package service

import (
	"sort"

	"couponcenter/internal/model"
)

// GetUsageRecord 查询使用记录详情。
func (s *Service) GetUsageRecord(id string) (*model.UsageRecord, error) {
	return s.store.GetUsageRecord(id)
}

// ListUsageRecords 分页查询使用记录，按核销时间倒序。
func (s *Service) ListUsageRecords(filter model.UsageFilter, page, size int) ([]*model.UsageRecord, int, error) {
	all := s.store.ListUsageRecords()
	matched := make([]*model.UsageRecord, 0, len(all))
	for _, u := range all {
		if filter.Match(u) {
			matched = append(matched, u)
		}
	}
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].UsedAt.After(matched[j].UsedAt)
	})
	total := len(matched)
	start := (page - 1) * size
	if start >= total {
		return []*model.UsageRecord{}, total, nil
	}
	end := start + size
	if end > total {
		end = total
	}
	return matched[start:end], total, nil
}
