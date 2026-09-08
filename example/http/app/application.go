package app

import (
	"github.com/pubgo/dix/v2"

	analyticshandler "github.com/pubgo/dix/example/http/domain/analytics/handler"
	analyticssvc "github.com/pubgo/dix/example/http/domain/analytics/service"
	billinghandler "github.com/pubgo/dix/example/http/domain/billing/handler"
	billingsvc "github.com/pubgo/dix/example/http/domain/billing/service"
	identityhandler "github.com/pubgo/dix/example/http/domain/identity/handler"
	identitysvc "github.com/pubgo/dix/example/http/domain/identity/service"
	inventoryhandler "github.com/pubgo/dix/example/http/domain/inventory/handler"
	inventorysvc "github.com/pubgo/dix/example/http/domain/inventory/service"
	mediahandler "github.com/pubgo/dix/example/http/domain/media/handler"
	mediasvc "github.com/pubgo/dix/example/http/domain/media/service"
	notifyhandler "github.com/pubgo/dix/example/http/domain/notification/handler"
	notifysvc "github.com/pubgo/dix/example/http/domain/notification/service"
	searchhandler "github.com/pubgo/dix/example/http/domain/searchx/handler"
	searchsvc "github.com/pubgo/dix/example/http/domain/searchx/service"
	shippinghandler "github.com/pubgo/dix/example/http/domain/shipping/handler"
	shippingsvc "github.com/pubgo/dix/example/http/domain/shipping/service"
	storagehandler "github.com/pubgo/dix/example/http/domain/storage/handler"
	storagesvc "github.com/pubgo/dix/example/http/domain/storage/service"
	workflowhandler "github.com/pubgo/dix/example/http/domain/workflow/handler"
	workflowsvc "github.com/pubgo/dix/example/http/domain/workflow/service"
	"github.com/pubgo/dix/example/http/infra/logger"
	"github.com/pubgo/dix/example/http/plugins"
)

// Application 应用聚合根:引用十个域服务 + 插件平台。
type Application struct {
	Logger    logger.Logger
	Billing   *billingsvc.Service
	Inventory *inventorysvc.Service
	Shipping  *shippingsvc.Service
	Identity  *identitysvc.Service
	Analytics *analyticssvc.Service
	Notify    *notifysvc.Service
	Search    *searchsvc.Service
	Storage   *storagesvc.Service
	Media     *mediasvc.Service
	Workflow  *workflowsvc.Service
	Plugins   []string
}

func Provide(di *dix.Dix) {
	dix.Provide(di, func(
		log logger.Logger,
		billingH *billinghandler.Handler,
		inventoryH *inventoryhandler.Handler,
		shippingH *shippinghandler.Handler,
		identityH *identityhandler.Handler,
		analyticsH *analyticshandler.Handler,
		notifyH *notifyhandler.Handler,
		searchH *searchhandler.Handler,
		storageH *storagehandler.Handler,
		mediaH *mediahandler.Handler,
		workflowH *workflowhandler.Handler,
		platform *plugins.Platform,
	) *Application {
		return &Application{
			Logger:    log,
			Billing:   billingH.Service,
			Inventory: inventoryH.Service,
			Shipping:  shippingH.Service,
			Identity:  identityH.Service,
			Analytics: analyticsH.Service,
			Notify:    notifyH.Service,
			Search:    searchH.Service,
			Storage:   storageH.Service,
			Media:     mediaH.Service,
			Workflow:  workflowH.Service,
			Plugins:   platform.Names,
		}
	})
}
