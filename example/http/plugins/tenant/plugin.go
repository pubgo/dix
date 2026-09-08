package tenant

import (
	"github.com/pubgo/dix/v2"

	"github.com/pubgo/dix/example/http/plugins"
)

type plugin struct{ name string }

func (p *plugin) Name() string { return p.name }

type worker struct{ name string }

func (w *worker) Name() string { return w.name }

func Provide(di *dix.Dix) {
	dix.Provide(di, func() map[string]plugins.Plugin {
		return map[string]plugins.Plugin{
			"tenant": &plugin{name: "tenant"},
		}
	})
	dix.Provide(di, func(all map[string]plugins.Plugin) map[string]plugins.Worker {
		p := all["tenant"]
		_ = p
		return map[string]plugins.Worker{
			"tenant": &worker{name: "tenant.worker"},
		}
	})
}
