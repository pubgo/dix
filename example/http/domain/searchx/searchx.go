// Package searchx 提供搜索域的示例组件:配置 → 客户端 → 仓储 → 服务 → 处理器
// 五层链路,外加多区域命名空间连接(演示对象规模与模块级视图)。
package searchx

import (
	"github.com/pubgo/dix/v2"
)

// Config 搜索配置。
type Config struct {
	Env     string
	Timeout string
}

// Client 搜索下游客户端。
type Client struct {
	Config *Config
}

// Repo 搜索仓储层。
type Repo struct {
	Client *Client
}

// Service 搜索服务层。
type Service struct {
	Repo *Repo
}

// Handler 搜索协议处理器。
type Handler struct {
	Service *Service
}

// Regions 多区域连接(命名空间 map:一次 provider 产出多个对象)。
type Regions map[string]*Client

// Providers 注册搜索域的全部 provider。
func Providers(di *dix.Dix) {
	dix.Provide(di, func() *Config {
		return &Config{Env: "prod", Timeout: "3s"}
	})
	dix.Provide(di, func(c *Config) *Client {
		return &Client{Config: c}
	})
	dix.Provide(di, func(c *Config) Regions {
		return Regions{
			"cn": {Config: c},
			"us": {Config: c},
			"eu": {Config: c},
		}
	})
	dix.Provide(di, func(c *Client) *Repo {
		return &Repo{Client: c}
	})
	dix.Provide(di, func(r *Repo) *Service {
		return &Service{Repo: r}
	})
	dix.Provide(di, func(s *Service) *Handler {
		return &Handler{Service: s}
	})
}
