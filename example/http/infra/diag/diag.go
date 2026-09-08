package diag

import (
	"errors"
	"log"
	"time"

	"github.com/pubgo/dix/v2"

	"github.com/pubgo/dix/example/http/infra/logger"
)

// SlowRemoteClient 模拟外部慢依赖(触发 provider_timeout)。
type SlowRemoteClient struct{ Ready bool }

// TimeoutProbe 触发 SlowRemoteClient 的解析(金字塔 entry #3)。
type TimeoutProbe struct{ Client *SlowRemoteClient }

// StartupMissingDependency 模拟注入缺失依赖。
type StartupMissingDependency struct{}

// StartupResolveInputMissing 模拟 provider 输入依赖缺失。
type StartupResolveInputMissing struct{}

// StartupResolveInputProbe 用于触发 StartupResolveInputMissing 的解析。
type StartupResolveInputProbe struct{}

// StartupBrokenComponent 模拟 provider 返回 error。
type StartupBrokenComponent struct{}

// StartupPanicComponent 模拟 provider panic。
type StartupPanicComponent struct{}

func Provide(di *dix.Dix) {
	dix.Provide(di, func(log logger.Logger) *SlowRemoteClient {
		log.Info("[demo] SlowRemoteClient start (expected timeout)")
		time.Sleep(450 * time.Millisecond)
		return &SlowRemoteClient{Ready: true}
	})
	dix.Provide(di, func(client *SlowRemoteClient) *TimeoutProbe {
		return &TimeoutProbe{Client: client}
	})
}

func logStartupScenarioResult(di *dix.Dix, scenario string, err error, previousCount int) {
	if err == nil {
		return
	}
	recent := di.GetRecentErrors(0)
	if len(recent) == 0 {
		log.Printf("⚠️ [startup-diagnostic][%s] err=%v (no recent error records)", scenario, err)
		return
	}
	newCount := len(recent) - previousCount
	if newCount <= 0 {
		newCount = 1
	}
	if newCount > len(recent) {
		newCount = len(recent)
	}
	log.Printf("⚠️ [startup-diagnostic][%s] captured %d record(s)", scenario, newCount)
	for i := newCount - 1; i >= 0; i-- {
		item := recent[i]
		log.Printf("   - record=%d type=%s op=%s message=%s",
			newCount-i, item.ErrorType, item.Operation, item.Message)
		if item.Hint != "" {
			log.Printf("     hint=%s", item.Hint)
		}
	}
}

// RunStartupErrorScenarios 在启动阶段统一触发多类可识别错误。
func RunStartupErrorScenarios(di *dix.Dix) {
	log.Println("🧪 Running startup error diagnostics...")

	before := len(di.GetRecentErrors(0))
	if err := di.TryProvide(nil); err != nil {
		logStartupScenarioResult(di, "invalid_provider_registration", err, before)
	}

	before = len(di.GetRecentErrors(0))
	if err := di.TryInject(func(*StartupMissingDependency) {}); err != nil {
		logStartupScenarioResult(di, "inject_missing_dependency", err, before)
	}

	{
		tmp := dix.New()
		logger.Provide(tmp)
		dix.Provide(tmp, func(*StartupResolveInputMissing) *StartupResolveInputProbe {
			return &StartupResolveInputProbe{}
		})
		before = len(tmp.GetRecentErrors(0))
		if err := tmp.TryInject(func(*StartupResolveInputProbe) {}); err != nil {
			logStartupScenarioResult(tmp, "provider_input_unresolved", err, before)
		}
	}

	{
		tmp := dix.New()
		logger.Provide(tmp)
		dix.Provide(tmp, func(log logger.Logger) (*StartupBrokenComponent, error) {
			log.Info("[demo] StartupBrokenComponent returns intentional error")
			return nil, errors.New("demo startup: provider return error")
		})
		before = len(tmp.GetRecentErrors(0))
		if err := tmp.TryInject(func(*StartupBrokenComponent) {}); err != nil {
			logStartupScenarioResult(tmp, "provider_return_error", err, before)
		}
	}

	before = len(di.GetRecentErrors(0))
	if err := di.TryInject(func(log logger.Logger) error {
		log.Info("[demo] inject callback returns intentional error")
		return errors.New("demo startup: inject callback error")
	}); err != nil {
		logStartupScenarioResult(di, "inject_callback_error", err, before)
	}

	{
		tmp := dix.New()
		logger.Provide(tmp)
		dix.Provide(tmp, func(log logger.Logger) *StartupPanicComponent {
			log.Info("[demo] StartupPanicComponent panics intentionally")
			panic("demo startup: provider panic")
		})
		before = len(tmp.GetRecentErrors(0))
		if err := tmp.TryInject(func(*StartupPanicComponent) {}); err != nil {
			logStartupScenarioResult(tmp, "provider_panic", err, before)
		}
	}

	before = len(di.GetRecentErrors(0))
	if err := di.TryInject(func(*TimeoutProbe) {}); err != nil {
		logStartupScenarioResult(di, "provider_timeout", err, before)
	}

	cycleDI := dix.New()
	type cycleA struct{}
	type cycleB struct{}
	type cycleC struct{}
	dix.Provide(cycleDI, func(*cycleC) *cycleA { return &cycleA{} })
	dix.Provide(cycleDI, func(*cycleA) *cycleB { return &cycleB{} })
	dix.Provide(cycleDI, func(*cycleB) *cycleC { return &cycleC{} })
	beforeCycle := len(cycleDI.GetRecentErrors(0))
	if err := cycleDI.TryInject(func(*cycleA) {}); err != nil {
		logStartupScenarioResult(cycleDI, "dependency_cycle(temp_container)", err, beforeCycle)
	}

	log.Println("🧪 Startup diagnostics done. Visit /api/errors to verify error_type/hint recognition.")
}
