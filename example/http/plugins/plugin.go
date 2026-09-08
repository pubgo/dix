package plugins

import "github.com/pubgo/dix/v2"

// Plugin 插件契约:实现方注册到 map[string]Plugin 命名空间。
type Plugin interface {
	Name() string
}

// Worker 工作器契约:依赖同名 Plugin 命名空间,对外仍只暴露接口。
type Worker interface {
	Name() string
}

// Platform 聚合全部 Worker,供 Application 消费(避免多入口)。
type Platform struct {
	WorkerCount int
	Names       []string
}

func Provide(di *dix.Dix) {
	dix.Provide(di, func(workers map[string]Worker) *Platform {
		names := make([]string, 0, len(workers)*2)
		for key, w := range workers {
			names = append(names, key)
			if w != nil {
				names = append(names, w.Name())
			}
		}
		return &Platform{
			WorkerCount: len(workers),
			Names:       names,
		}
	})
}
