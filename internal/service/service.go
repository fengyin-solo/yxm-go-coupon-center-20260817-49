// Package service 实现业务逻辑层。
package service

import (
	"couponcenter/internal/config"
	"couponcenter/internal/model"
	"couponcenter/internal/store"
	"couponcenter/pkg/logger"
)

// Service 聚合全部业务方法，依赖 Store 做数据访问。
type Service struct {
	store store.Store
	log   *logger.Logger
	cfg   *config.Config
}

func New(st store.Store, log *logger.Logger, cfg *config.Config) *Service {
	return &Service{store: st, log: log, cfg: cfg}
}

func normalizePage(page, size int) error {
	if page <= 0 {
		return model.NewValidationError("page", "页码必须大于 0")
	}
	if size <= 0 {
		return model.NewValidationError("size", "每页数量必须大于 0")
	}
	return nil
}
