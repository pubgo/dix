// Package analytics 提供数据分析域的示例组件:配置 → 客户端 → 仓储 → 服务 → 处理器
// 五层链路,外加多区域命名空间连接(演示对象规模与模块级视图)。
package analytics

import (
	"github.com/pubgo/dix/v2"
)

// Config 数据分析配置。
type Config struct {
	Env     string
	Timeout string
}

// Client 数据分析下游客户端。
type Client struct {
	Config *Config
}

// Repo 数据分析仓储层。
type Repo struct {
	Client *Client
}

// Service 数据分析服务层。
type Service struct {
	Repo *Repo
}

// Handler 数据分析协议处理器。
type Handler struct {
	Service *Service
}

// Regions 多区域连接(命名空间 map:一次 provider 产出多个对象)。
type Regions map[string]*Client

// Providers 注册数据分析域的全部 provider。
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
