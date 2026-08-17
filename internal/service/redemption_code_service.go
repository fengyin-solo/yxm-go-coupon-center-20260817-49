package service

import (
	"sort"
	"time"

	"couponcenter/internal/model"
	"couponcenter/internal/store"
	"couponcenter/pkg/idgen"
)

// CreateRedeemCode 创建单个兑换码，关联模板必须存在。
func (s *Service) CreateRedeemCode(templateID, code string, expireAt time.Time) (*model.RedemptionCode, error) {
	if _, err := s.store.GetTemplate(templateID); err != nil {
		return nil, model.NewValidationError("template_id", "关联模板不存在")
	}
	rc := &model.RedemptionCode{
		ID:         idgen.Hex(),
		Code:       code,
		TemplateID: templateID,
		Status:     model.RedeemCodeAvailable,
		ExpireAt:   expireAt,
		CreatedAt:  time.Now(),
	}
	if err := rc.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.CreateRedeemCode(rc); err != nil {
		return nil, err
	}
	s.log.Infof("创建兑换码 %s（模板 %s）", rc.Code, templateID)
	return rc, nil
}

// BatchCreateRedeemCodes 批量生成兑换码，码值基于短码自动产生。
// 返回成功创建的数量；若模板不存在则报错。
func (s *Service) BatchCreateRedeemCodes(templateID string, count int, expireAt time.Time) (int, error) {
	if count <= 0 || count > 1000 {
		return 0, model.NewValidationError("count", "批量数量需在 1-1000 之间")
	}
	if _, err := s.store.GetTemplate(templateID); err != nil {
		return 0, model.NewValidationError("template_id", "关联模板不存在")
	}
	created := 0
	now := time.Now()
	for i := 0; i < count; i++ {
		rc := &model.RedemptionCode{
			ID:         idgen.Hex(),
			Code:       "RC-" + idgen.Short() + idgen.HexN(3),
			TemplateID: templateID,
			Status:     model.RedeemCodeAvailable,
			ExpireAt:   expireAt,
			CreatedAt:  now,
		}
		if err := rc.Validate(); err != nil {
			continue
		}
		if err := s.store.CreateRedeemCode(rc); err != nil {
			continue
		}
		created++
	}
	s.log.Infof("批量创建兑换码：请求 %d 个，成功 %d 个", count, created)
	return created, nil
}

// GetRedeemCode 查询兑换码详情。
func (s *Service) GetRedeemCode(id string) (*model.RedemptionCode, error) {
	return s.store.GetRedeemCode(id)
}

// ListRedeemCodes 分页查询兑换码列表。
func (s *Service) ListRedeemCodes(filter model.RedeemCodeFilter, page, size int) ([]*model.RedemptionCode, int, error) {
	all := s.store.ListRedeemCodes()
	matched := make([]*model.RedemptionCode, 0, len(all))
	for _, c := range all {
		if filter.Match(c) {
			matched = append(matched, c)
		}
	}
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].CreatedAt.After(matched[j].CreatedAt)
	})
	total := len(matched)
	start := (page - 1) * size
	if start >= total {
		return []*model.RedemptionCode{}, total, nil
	}
	end := start + size
	if end > total {
		end = total
	}
	return matched[start:end], total, nil
}

// RedeemByCode 用户输入兑换码领取优惠券。
// 校验链：码存在 -> 状态 available -> 未过期 -> 走标准领券流程 -> 码置为 redeemed。
func (s *Service) RedeemByCode(code, userID string) (*model.UserCoupon, error) {
	rc, err := s.store.GetRedeemCodeByCode(code)
	if err != nil {
		return nil, err
	}
	if rc.Status != model.RedeemCodeAvailable {
		return nil, model.NewValidationError("code", "兑换码已被使用或已作废")
	}
	if rc.IsExpired(time.Now()) {
		return nil, model.NewValidationError("code", "兑换码已过期")
	}
	coupon, err := s.ClaimCoupon(rc.TemplateID, userID)
	if err != nil {
		return nil, err
	}
	rc.RedeemedAt = time.Now()
	if err := s.store.UpdateRedeemCode(rc); err != nil {
		return nil, err
	}
	s.log.Infof("兑换码 %s 被用户 %s 兑换", code, userID)
	return coupon, nil
}

// VoidRedeemCode 作废一个未使用的兑换码。
func (s *Service) VoidRedeemCode(id string) (*model.RedemptionCode, error) {
	rc, err := s.store.GetRedeemCode(id)
	if err != nil {
		return nil, err
	}
	if !model.CanRedeemCodeTransition(rc.Status, model.RedeemCodeVoid) {
		return nil, model.NewValidationError("status", "当前状态不可作废")
	}
	rc.Status = model.RedeemCodeVoid
	if err := s.store.UpdateRedeemCode(rc); err != nil {
		return nil, err
	}
	s.log.Infof("兑换码 %s 已作废", rc.Code)
	return rc, nil
}

// DeleteRedeemCode 删除兑换码；已兑换的码不允许删除。
func (s *Service) DeleteRedeemCode(id string) error {
	rc, err := s.store.GetRedeemCode(id)
	if err != nil {
		return err
	}
	if rc.Status == model.RedeemCodeRedeemed {
		return store.ErrConflict
	}
	return s.store.DeleteRedeemCode(id)
}
