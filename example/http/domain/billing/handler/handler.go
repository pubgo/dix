package handler

import (
	"github.com/pubgo/dix/v2"

	dinfr "github.com/pubgo/dix/example/http/domain/billing/infra"
	"github.com/pubgo/dix/example/http/domain/billing/service"
)

// Handler billing 协议处理器。
type Handler struct {
	Service *service.Service
}

func Provide(di *dix.Dix) {
	dix.Provide(di, func(s *service.Service, regions dinfr.Regions) *Handler {
		_ = regions
		return &Handler{Service: s}
	})
}
