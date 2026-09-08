package infra

import (
	"github.com/pubgo/dix/v2"

	"github.com/pubgo/dix/example/http/domain/storage/models"
)

// Client storage 下游客户端。
type Client struct {
	Config *models.Config
}

// Regions 多区域连接。
type Regions map[string]*Client

func Provide(di *dix.Dix) {
	dix.Provide(di, func(c *models.Config) *Client {
		return &Client{Config: c}
	})
	dix.Provide(di, func(c *models.Config) Regions {
		return Regions{
			"cn": {Config: c},
			"us": {Config: c},
			"eu": {Config: c},
		}
	})
}
