package main

// 本文件为批量合成的"插件/工作器"依赖族:用泛型实例化在编译期产出
// 大量相互独立的依赖类型,把演示容器撑到真实项目规模。

import "github.com/pubgo/dix/v2"

// Plugin 是第一族合成依赖的载体。
type Plugin[T any] struct {
	Name    string
	Version string
}

// Worker 是第二族合成依赖的载体,依赖同角色的 Plugin(形成链路)。
type Worker[T any] struct {
	Name  string
	Batch int
}

type (
	RoleAuthReader       struct{}
	RoleAuthWriter       struct{}
	RoleAuthValidator    struct{}
	RoleBillingReader    struct{}
	RoleBillingWriter    struct{}
	RoleBillingValidator struct{}
	RoleCacheReader      struct{}
	RoleCacheWriter      struct{}
	RoleCacheValidator   struct{}
	RoleEmailReader      struct{}
	RoleEmailWriter      struct{}
	RoleEmailValidator   struct{}
	RoleExportReader     struct{}
	RoleExportWriter     struct{}
	RoleExportValidator  struct{}
	RoleGraphqlReader    struct{}
	RoleGraphqlWriter    struct{}
	RoleGraphqlValidator struct{}
	RoleImportReader     struct{}
	RoleImportWriter     struct{}
	RoleImportValidator  struct{}
	RoleJobReader        struct{}
	RoleJobWriter        struct{}
	RoleJobValidator     struct{}
	RoleKafkaReader      struct{}
	RoleKafkaWriter      struct{}
	RoleKafkaValidator   struct{}
	RoleLoginReader      struct{}
	RoleLoginWriter      struct{}
	RoleLoginValidator   struct{}
	RoleMetricsReader    struct{}
	RoleMetricsWriter    struct{}
	RoleMetricsValidator struct{}
	RoleNotifyReader     struct{}
	RoleNotifyWriter     struct{}
	RoleNotifyValidator  struct{}
	RoleOauthReader      struct{}
	RoleOauthWriter      struct{}
	RoleOauthValidator   struct{}
	RoleQueueReader      struct{}
	RoleQueueWriter      struct{}
	RoleQueueValidator   struct{}
	RoleReportReader     struct{}
	RoleReportWriter     struct{}
	RoleReportValidator  struct{}
	RoleSearchReader     struct{}
	RoleSearchWriter     struct{}
	RoleSearchValidator  struct{}
	RoleSessionReader    struct{}
	RoleSessionWriter    struct{}
	RoleSessionValidator struct{}
	RoleTenantReader     struct{}
	RoleTenantWriter     struct{}
	RoleTenantValidator  struct{}
	RoleUploadReader     struct{}
	RoleUploadWriter     struct{}
	RoleUploadValidator  struct{}
	RoleVaultReader      struct{}
	RoleVaultWriter      struct{}
	RoleVaultValidator   struct{}
)

var pluginNames []string

// registerPlugin 注册一个角色插件 provider,返回用于预创建对象的激活函数。
func registerPlugin[T any](di *dix.Dix, name string) func() {
	dix.Provide(di, func() *Plugin[T] {
		pluginNames = append(pluginNames, name)
		return &Plugin[T]{Name: name, Version: "1.0.0"}
	})
	return func() { _ = dix.Inject(di, func(p *Plugin[T]) {}) }
}

// registerWorker 注册依赖同角色插件的工作器 provider。
func registerWorker[T any](di *dix.Dix, name string) func() {
	dix.Provide(di, func(p *Plugin[T]) *Worker[T] {
		pluginNames = append(pluginNames, name+".worker")
		return &Worker[T]{Name: name, Batch: 8}
	})
	return func() { _ = dix.Inject(di, func(w *Worker[T]) {}) }
}

// registerPlugins 批量注册角色插件与工作器,返回全部激活函数。
func registerPlugins(di *dix.Dix) (activators []func()) {
	for _, r := range []struct {
		name string
		reg  func(*dix.Dix, string) func()
	}{
		{"authreader", registerPlugin[RoleAuthReader]},
		{"authreader.worker", registerWorker[RoleAuthReader]},
		{"authwriter", registerPlugin[RoleAuthWriter]},
		{"authwriter.worker", registerWorker[RoleAuthWriter]},
		{"authvalidator", registerPlugin[RoleAuthValidator]},
		{"authvalidator.worker", registerWorker[RoleAuthValidator]},
		{"billingreader", registerPlugin[RoleBillingReader]},
		{"billingreader.worker", registerWorker[RoleBillingReader]},
		{"billingwriter", registerPlugin[RoleBillingWriter]},
		{"billingwriter.worker", registerWorker[RoleBillingWriter]},
		{"billingvalidator", registerPlugin[RoleBillingValidator]},
		{"billingvalidator.worker", registerWorker[RoleBillingValidator]},
		{"cachereader", registerPlugin[RoleCacheReader]},
		{"cachereader.worker", registerWorker[RoleCacheReader]},
		{"cachewriter", registerPlugin[RoleCacheWriter]},
		{"cachewriter.worker", registerWorker[RoleCacheWriter]},
		{"cachevalidator", registerPlugin[RoleCacheValidator]},
		{"cachevalidator.worker", registerWorker[RoleCacheValidator]},
		{"emailreader", registerPlugin[RoleEmailReader]},
		{"emailreader.worker", registerWorker[RoleEmailReader]},
		{"emailwriter", registerPlugin[RoleEmailWriter]},
		{"emailwriter.worker", registerWorker[RoleEmailWriter]},
		{"emailvalidator", registerPlugin[RoleEmailValidator]},
		{"emailvalidator.worker", registerWorker[RoleEmailValidator]},
		{"exportreader", registerPlugin[RoleExportReader]},
		{"exportreader.worker", registerWorker[RoleExportReader]},
		{"exportwriter", registerPlugin[RoleExportWriter]},
		{"exportwriter.worker", registerWorker[RoleExportWriter]},
		{"exportvalidator", registerPlugin[RoleExportValidator]},
		{"exportvalidator.worker", registerWorker[RoleExportValidator]},
		{"graphqlreader", registerPlugin[RoleGraphqlReader]},
		{"graphqlreader.worker", registerWorker[RoleGraphqlReader]},
		{"graphqlwriter", registerPlugin[RoleGraphqlWriter]},
		{"graphqlwriter.worker", registerWorker[RoleGraphqlWriter]},
		{"graphqlvalidator", registerPlugin[RoleGraphqlValidator]},
		{"graphqlvalidator.worker", registerWorker[RoleGraphqlValidator]},
		{"importreader", registerPlugin[RoleImportReader]},
		{"importreader.worker", registerWorker[RoleImportReader]},
		{"importwriter", registerPlugin[RoleImportWriter]},
		{"importwriter.worker", registerWorker[RoleImportWriter]},
		{"importvalidator", registerPlugin[RoleImportValidator]},
		{"importvalidator.worker", registerWorker[RoleImportValidator]},
		{"jobreader", registerPlugin[RoleJobReader]},
		{"jobreader.worker", registerWorker[RoleJobReader]},
		{"jobwriter", registerPlugin[RoleJobWriter]},
		{"jobwriter.worker", registerWorker[RoleJobWriter]},
		{"jobvalidator", registerPlugin[RoleJobValidator]},
		{"jobvalidator.worker", registerWorker[RoleJobValidator]},
		{"kafkareader", registerPlugin[RoleKafkaReader]},
		{"kafkareader.worker", registerWorker[RoleKafkaReader]},
		{"kafkawriter", registerPlugin[RoleKafkaWriter]},
		{"kafkawriter.worker", registerWorker[RoleKafkaWriter]},
		{"kafkavalidator", registerPlugin[RoleKafkaValidator]},
		{"kafkavalidator.worker", registerWorker[RoleKafkaValidator]},
		{"loginreader", registerPlugin[RoleLoginReader]},
		{"loginreader.worker", registerWorker[RoleLoginReader]},
		{"loginwriter", registerPlugin[RoleLoginWriter]},
		{"loginwriter.worker", registerWorker[RoleLoginWriter]},
		{"loginvalidator", registerPlugin[RoleLoginValidator]},
		{"loginvalidator.worker", registerWorker[RoleLoginValidator]},
		{"metricsreader", registerPlugin[RoleMetricsReader]},
		{"metricsreader.worker", registerWorker[RoleMetricsReader]},
		{"metricswriter", registerPlugin[RoleMetricsWriter]},
		{"metricswriter.worker", registerWorker[RoleMetricsWriter]},
		{"metricsvalidator", registerPlugin[RoleMetricsValidator]},
		{"metricsvalidator.worker", registerWorker[RoleMetricsValidator]},
		{"notifyreader", registerPlugin[RoleNotifyReader]},
		{"notifyreader.worker", registerWorker[RoleNotifyReader]},
		{"notifywriter", registerPlugin[RoleNotifyWriter]},
		{"notifywriter.worker", registerWorker[RoleNotifyWriter]},
		{"notifyvalidator", registerPlugin[RoleNotifyValidator]},
		{"notifyvalidator.worker", registerWorker[RoleNotifyValidator]},
		{"oauthreader", registerPlugin[RoleOauthReader]},
		{"oauthreader.worker", registerWorker[RoleOauthReader]},
		{"oauthwriter", registerPlugin[RoleOauthWriter]},
		{"oauthwriter.worker", registerWorker[RoleOauthWriter]},
		{"oauthvalidator", registerPlugin[RoleOauthValidator]},
		{"oauthvalidator.worker", registerWorker[RoleOauthValidator]},
		{"queuereader", registerPlugin[RoleQueueReader]},
		{"queuereader.worker", registerWorker[RoleQueueReader]},
		{"queuewriter", registerPlugin[RoleQueueWriter]},
		{"queuewriter.worker", registerWorker[RoleQueueWriter]},
		{"queuevalidator", registerPlugin[RoleQueueValidator]},
		{"queuevalidator.worker", registerWorker[RoleQueueValidator]},
		{"reportreader", registerPlugin[RoleReportReader]},
		{"reportreader.worker", registerWorker[RoleReportReader]},
		{"reportwriter", registerPlugin[RoleReportWriter]},
		{"reportwriter.worker", registerWorker[RoleReportWriter]},
		{"reportvalidator", registerPlugin[RoleReportValidator]},
		{"reportvalidator.worker", registerWorker[RoleReportValidator]},
		{"searchreader", registerPlugin[RoleSearchReader]},
		{"searchreader.worker", registerWorker[RoleSearchReader]},
		{"searchwriter", registerPlugin[RoleSearchWriter]},
		{"searchwriter.worker", registerWorker[RoleSearchWriter]},
		{"searchvalidator", registerPlugin[RoleSearchValidator]},
		{"searchvalidator.worker", registerWorker[RoleSearchValidator]},
		{"sessionreader", registerPlugin[RoleSessionReader]},
		{"sessionreader.worker", registerWorker[RoleSessionReader]},
		{"sessionwriter", registerPlugin[RoleSessionWriter]},
		{"sessionwriter.worker", registerWorker[RoleSessionWriter]},
		{"sessionvalidator", registerPlugin[RoleSessionValidator]},
		{"sessionvalidator.worker", registerWorker[RoleSessionValidator]},
		{"tenantreader", registerPlugin[RoleTenantReader]},
		{"tenantreader.worker", registerWorker[RoleTenantReader]},
		{"tenantwriter", registerPlugin[RoleTenantWriter]},
		{"tenantwriter.worker", registerWorker[RoleTenantWriter]},
		{"tenantvalidator", registerPlugin[RoleTenantValidator]},
		{"tenantvalidator.worker", registerWorker[RoleTenantValidator]},
		{"uploadreader", registerPlugin[RoleUploadReader]},
		{"uploadreader.worker", registerWorker[RoleUploadReader]},
		{"uploadwriter", registerPlugin[RoleUploadWriter]},
		{"uploadwriter.worker", registerWorker[RoleUploadWriter]},
		{"uploadvalidator", registerPlugin[RoleUploadValidator]},
		{"uploadvalidator.worker", registerWorker[RoleUploadValidator]},
		{"vaultreader", registerPlugin[RoleVaultReader]},
		{"vaultreader.worker", registerWorker[RoleVaultReader]},
		{"vaultwriter", registerPlugin[RoleVaultWriter]},
		{"vaultwriter.worker", registerWorker[RoleVaultWriter]},
		{"vaultvalidator", registerPlugin[RoleVaultValidator]},
		{"vaultvalidator.worker", registerWorker[RoleVaultValidator]},
	} {
		activators = append(activators, r.reg(di, r.name))
	}
	return activators
}
