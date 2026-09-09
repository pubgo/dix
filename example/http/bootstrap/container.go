package bootstrap

import (
	"log"
	"net/http"
	"time"

	"github.com/pubgo/dix/v2"
	"github.com/pubgo/dix/v2/dixhttp"

	"github.com/pubgo/dix/example/http/app"
	analyticshandler "github.com/pubgo/dix/example/http/domain/analytics/handler"
	analyticsinfra "github.com/pubgo/dix/example/http/domain/analytics/infra"
	analyticslogic "github.com/pubgo/dix/example/http/domain/analytics/logic"
	analyticsmodels "github.com/pubgo/dix/example/http/domain/analytics/models"
	analyticssvc "github.com/pubgo/dix/example/http/domain/analytics/service"
	billinghandler "github.com/pubgo/dix/example/http/domain/billing/handler"
	billinginfra "github.com/pubgo/dix/example/http/domain/billing/infra"
	billinglogic "github.com/pubgo/dix/example/http/domain/billing/logic"
	billingmodels "github.com/pubgo/dix/example/http/domain/billing/models"
	billingsvc "github.com/pubgo/dix/example/http/domain/billing/service"
	identityhandler "github.com/pubgo/dix/example/http/domain/identity/handler"
	identityinfra "github.com/pubgo/dix/example/http/domain/identity/infra"
	identitylogic "github.com/pubgo/dix/example/http/domain/identity/logic"
	identitymodels "github.com/pubgo/dix/example/http/domain/identity/models"
	identitysvc "github.com/pubgo/dix/example/http/domain/identity/service"
	inventoryhandler "github.com/pubgo/dix/example/http/domain/inventory/handler"
	inventoryinfra "github.com/pubgo/dix/example/http/domain/inventory/infra"
	inventorylogic "github.com/pubgo/dix/example/http/domain/inventory/logic"
	inventorymodels "github.com/pubgo/dix/example/http/domain/inventory/models"
	inventorysvc "github.com/pubgo/dix/example/http/domain/inventory/service"
	mediahandler "github.com/pubgo/dix/example/http/domain/media/handler"
	mediainfra "github.com/pubgo/dix/example/http/domain/media/infra"
	medialogic "github.com/pubgo/dix/example/http/domain/media/logic"
	mediamodels "github.com/pubgo/dix/example/http/domain/media/models"
	mediasvc "github.com/pubgo/dix/example/http/domain/media/service"
	notifyhandler "github.com/pubgo/dix/example/http/domain/notification/handler"
	notifyinfra "github.com/pubgo/dix/example/http/domain/notification/infra"
	notifylogic "github.com/pubgo/dix/example/http/domain/notification/logic"
	notifymodels "github.com/pubgo/dix/example/http/domain/notification/models"
	notifysvc "github.com/pubgo/dix/example/http/domain/notification/service"
	searchhandler "github.com/pubgo/dix/example/http/domain/searchx/handler"
	searchinfra "github.com/pubgo/dix/example/http/domain/searchx/infra"
	searchlogic "github.com/pubgo/dix/example/http/domain/searchx/logic"
	searchmodels "github.com/pubgo/dix/example/http/domain/searchx/models"
	searchsvc "github.com/pubgo/dix/example/http/domain/searchx/service"
	shippinghandler "github.com/pubgo/dix/example/http/domain/shipping/handler"
	shippinginfra "github.com/pubgo/dix/example/http/domain/shipping/infra"
	shippinglogic "github.com/pubgo/dix/example/http/domain/shipping/logic"
	shippingmodels "github.com/pubgo/dix/example/http/domain/shipping/models"
	shippingsvc "github.com/pubgo/dix/example/http/domain/shipping/service"
	storagehandler "github.com/pubgo/dix/example/http/domain/storage/handler"
	storageinfra "github.com/pubgo/dix/example/http/domain/storage/infra"
	storagelogic "github.com/pubgo/dix/example/http/domain/storage/logic"
	storagemodels "github.com/pubgo/dix/example/http/domain/storage/models"
	storagesvc "github.com/pubgo/dix/example/http/domain/storage/service"
	workflowhandler "github.com/pubgo/dix/example/http/domain/workflow/handler"
	workflowinfra "github.com/pubgo/dix/example/http/domain/workflow/infra"
	workflowlogic "github.com/pubgo/dix/example/http/domain/workflow/logic"
	workflowmodels "github.com/pubgo/dix/example/http/domain/workflow/models"
	workflowsvc "github.com/pubgo/dix/example/http/domain/workflow/service"
	"github.com/pubgo/dix/example/http/infra/diag"
	"github.com/pubgo/dix/example/http/infra/logger"
	"github.com/pubgo/dix/example/http/plugins"
	"github.com/pubgo/dix/example/http/plugins/auth"
	pluginbilling "github.com/pubgo/dix/example/http/plugins/billing"
	"github.com/pubgo/dix/example/http/plugins/cache"
	"github.com/pubgo/dix/example/http/plugins/email"
	"github.com/pubgo/dix/example/http/plugins/export"
	"github.com/pubgo/dix/example/http/plugins/graphql"
	"github.com/pubgo/dix/example/http/plugins/importx"
	"github.com/pubgo/dix/example/http/plugins/job"
	"github.com/pubgo/dix/example/http/plugins/kafka"
	"github.com/pubgo/dix/example/http/plugins/login"
	"github.com/pubgo/dix/example/http/plugins/metrics"
	pluginnotify "github.com/pubgo/dix/example/http/plugins/notify"
	"github.com/pubgo/dix/example/http/plugins/oauth"
	"github.com/pubgo/dix/example/http/plugins/queue"
	"github.com/pubgo/dix/example/http/plugins/report"
	pluginsearch "github.com/pubgo/dix/example/http/plugins/search"
	"github.com/pubgo/dix/example/http/plugins/session"
	"github.com/pubgo/dix/example/http/plugins/tenant"
	"github.com/pubgo/dix/example/http/plugins/upload"
	"github.com/pubgo/dix/example/http/plugins/vault"
	"github.com/pubgo/dix/example/http/router"
)

// Build 装配完整演示容器:各模块自行 Provide,本包只负责编排顺序。
func Build() *dix.Dix {
	di := dix.New(
		dix.WithProviderTimeout(200*time.Millisecond),
		dix.WithSlowProviderThreshold(80*time.Millisecond),
	)

	logger.Provide(di)

	analyticsmodels.Provide(di)
	analyticsinfra.Provide(di)
	analyticslogic.Provide(di)
	analyticssvc.Provide(di)
	analyticshandler.Provide(di)

	billingmodels.Provide(di)
	billinginfra.Provide(di)
	billinglogic.Provide(di)
	billingsvc.Provide(di)
	billinghandler.Provide(di)

	identitymodels.Provide(di)
	identityinfra.Provide(di)
	identitylogic.Provide(di)
	identitysvc.Provide(di)
	identityhandler.Provide(di)

	inventorymodels.Provide(di)
	inventoryinfra.Provide(di)
	inventorylogic.Provide(di)
	inventorysvc.Provide(di)
	inventoryhandler.Provide(di)

	mediamodels.Provide(di)
	mediainfra.Provide(di)
	medialogic.Provide(di)
	mediasvc.Provide(di)
	mediahandler.Provide(di)

	notifymodels.Provide(di)
	notifyinfra.Provide(di)
	notifylogic.Provide(di)
	notifysvc.Provide(di)
	notifyhandler.Provide(di)

	searchmodels.Provide(di)
	searchinfra.Provide(di)
	searchlogic.Provide(di)
	searchsvc.Provide(di)
	searchhandler.Provide(di)

	shippingmodels.Provide(di)
	shippinginfra.Provide(di)
	shippinglogic.Provide(di)
	shippingsvc.Provide(di)
	shippinghandler.Provide(di)

	storagemodels.Provide(di)
	storageinfra.Provide(di)
	storagelogic.Provide(di)
	storagesvc.Provide(di)
	storagehandler.Provide(di)

	workflowmodels.Provide(di)
	workflowinfra.Provide(di)
	workflowlogic.Provide(di)
	workflowsvc.Provide(di)
	workflowhandler.Provide(di)

	auth.Provide(di)
	pluginbilling.Provide(di)
	cache.Provide(di)
	email.Provide(di)
	export.Provide(di)
	graphql.Provide(di)
	importx.Provide(di)
	job.Provide(di)
	kafka.Provide(di)
	login.Provide(di)
	metrics.Provide(di)
	pluginnotify.Provide(di)
	oauth.Provide(di)
	queue.Provide(di)
	report.Provide(di)
	pluginsearch.Provide(di)
	session.Provide(di)
	tenant.Provide(di)
	upload.Provide(di)
	vault.Provide(di)
	plugins.Provide(di)

	app.Provide(di)
	diag.Provide(di)

	return di
}

func preCreateObjects(di *dix.Dix) {
	if err := di.TryInject(func(a *app.Application) {
		log.Printf("✅ Application created: modules=10 plugins=%d", len(a.Plugins))
	}); err != nil {
		log.Printf("⚠️ pre-create injection failed: %v", err)
	}
	_ = di.TryInject(func(workers map[string]plugins.Worker) {
		_ = workers
	})
}

// Run 构建容器、预热对象、跑启动诊断并启动可视化 HTTP。
func Run() error {
	di := Build()
	preCreateObjects(di)
	diag.RunStartupErrorScenarios(di)
	log.Println("")
	server := dixhttp.NewServer(di)
	if err := router.StartVisualizationServer(server); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
