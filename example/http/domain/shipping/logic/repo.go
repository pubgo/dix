package logic

import (
	"github.com/pubgo/dix/v2"

	"github.com/pubgo/dix/example/http/domain/shipping/infra"
)

// Repo shipping 仓储 / 领域逻辑。
type Repo struct {
	Client *infra.Client
}

func Provide(di *dix.Dix) {
	dix.Provide(di, func(c *infra.Client) *Repo {
		return &Repo{Client: c}
	})
}
