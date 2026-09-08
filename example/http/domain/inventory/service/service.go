package service

import (
	"github.com/pubgo/dix/v2"

	"github.com/pubgo/dix/example/http/domain/inventory/logic"
)

// Service inventory 应用服务。
type Service struct {
	Repo *logic.Repo
}

func Provide(di *dix.Dix) {
	dix.Provide(di, func(r *logic.Repo) *Service {
		return &Service{Repo: r}
	})
}
