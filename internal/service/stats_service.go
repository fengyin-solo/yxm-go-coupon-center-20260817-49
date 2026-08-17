package service

import (
	"sort"

	"couponcenter/internal/model"
)

// OverviewStats 全局概览统计。
type OverviewStats struct {
	TemplateCount  int   `json:"template_count"`
	IssuedCount    int   `json:"issued_count"`    // 累计领券数
	UsedCount      int   `json:"used_count"`      // 累计核销数
	ExpiredCount   int   `json:"expired_count"`   // 过期券数
	TotalDiscount  int64 `json:"total_discount"`  // 累计抵扣金额（分）
	TotalOrderAmt  int64 `json:"total_order_amt"` // 核销订单总金额（分）
	RedemptionRate int   `json:"redemption_rate"` // 核销率（百分比，四舍五入）
}

// Stats 全局概览：领券、核销、抵扣总额、核销率。
func (s *Service) Stats() *OverviewStats {
	coupons := s.store.ListUserCoupons()
	records := s.store.ListUsageRecords()

	stats := &OverviewStats{
		TemplateCount: len(s.store.ListTemplates()),
		IssuedCount:   len(coupons),
	}
	for _, c := range coupons {
		switch c.Status {
		case model.UserCouponUsed:
			stats.UsedCount++
		case model.UserCouponExpired:
			stats.ExpiredCount++
		}
	}
	for _, u := range records {
		stats.TotalDiscount += u.DiscountAmt
		stats.TotalOrderAmt += u.OrderAmount
	}
	if stats.IssuedCount > 0 {
		stats.RedemptionRate = (stats.UsedCount*100 + stats.IssuedCount/2) / stats.IssuedCount
	}
	return stats
}

// TemplateStat 单模板统计。
type TemplateStat struct {
	TemplateID  string `json:"template_id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	IssuedCount int    `json:"issued_count"`
	UsedCount   int    `json:"used_count"`
	DiscountSum int64  `json:"discount_sum"` // 该模板累计抵扣（分）
}

// StatsByTemplate 按模板分组统计核销情况，按抵扣金额降序。
func (s *Service) StatsByTemplate() []*TemplateStat {
	byTemplate := make(map[string]*TemplateStat)
	for _, t := range s.store.ListTemplates() {
		byTemplate[t.ID] = &TemplateStat{
			TemplateID:  t.ID,
			Code:        t.Code,
			Name:        t.Name,
			IssuedCount: t.IssuedCount,
		}
	}
	for _, c := range s.store.ListUserCoupons() {
		if st, ok := byTemplate[c.TemplateID]; ok && c.Status == model.UserCouponUsed {
			st.UsedCount++
		}
	}
	for _, u := range s.store.ListUsageRecords() {
		if st, ok := byTemplate[u.TemplateID]; ok {
			st.DiscountSum += u.DiscountAmt
		}
	}
	list := make([]*TemplateStat, 0, len(byTemplate))
	for _, st := range byTemplate {
		list = append(list, st)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].DiscountSum != list[j].DiscountSum {
			return list[i].DiscountSum > list[j].DiscountSum
		}
		return list[i].Code < list[j].Code
	})
	return list
}

// DailyStat 单日核销统计。
type DailyStat struct {
	Date        string `json:"date"` // YYYY-MM-DD
	UsedCount   int    `json:"used_count"`
	DiscountSum int64  `json:"discount_sum"`
	OrderAmtSum int64  `json:"order_amt_sum"`
}

// StatsByDay 按天分组统计核销，按日期升序。
func (s *Service) StatsByDay() []*DailyStat {
	byDay := make(map[string]*DailyStat)
	for _, u := range s.store.ListUsageRecords() {
		day := u.UsedAt.Format("2006-01-02")
		st, ok := byDay[day]
		if !ok {
			st = &DailyStat{Date: day}
			byDay[day] = st
		}
		st.UsedCount++
		st.DiscountSum += u.DiscountAmt
		st.OrderAmtSum += u.OrderAmount
	}
	list := make([]*DailyStat, 0, len(byDay))
	for _, st := range byDay {
		list = append(list, st)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Date < list[j].Date })
	return list
}

// TopUser 用户核销排行条目。
type TopUser struct {
	UserID      string `json:"user_id"`
	UsedCount   int    `json:"used_count"`
	DiscountSum int64  `json:"discount_sum"`
}

// TopUsers 按核销抵扣金额取 TOP N 用户。
func (s *Service) TopUsers(n int) []*TopUser {
	if n <= 0 {
		n = 10
	}
	byUser := make(map[string]*TopUser)
	for _, u := range s.store.ListUsageRecords() {
		st, ok := byUser[u.UserID]
		if !ok {
			st = &TopUser{UserID: u.UserID}
			byUser[u.UserID] = st
		}
		st.UsedCount++
		st.DiscountSum += u.DiscountAmt
	}
	list := make([]*TopUser, 0, len(byUser))
	for _, st := range byUser {
		list = append(list, st)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].DiscountSum != list[j].DiscountSum {
			return list[i].DiscountSum > list[j].DiscountSum
		}
		return list[i].UserID < list[j].UserID
	})
	if len(list) > n {
		list = list[:n]
	}
	return list
}
