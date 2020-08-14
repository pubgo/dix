package log_test

import (
	"fmt"
	"github.com/pubgo/dix"
	"github.com/pubgo/xerror"
	"github.com/pubgo/xlog"
	"github.com/pubgo/xlog/xlog_config"
	"testing"
	"time"
)

var log = xlog.GetDevLog()

func init() {
	dix.Go(func(log1 xlog.XLog) {
		log = log1.
			Named("service").With(xlog.String("key", "value1")).
			Named("hello").With(xlog.String("key", "value2")).
			Named("world").With(xlog.String("key", "value3"))
	})
}

func TestExample(t *testing.T) {
	for {
		log.Debug("hello",
			xlog.Any("hss", "ss"),
		)

		log.Info("hello",
			xlog.Any("hss", "ss"),
		)
		fmt.Println(dix.Graph())
		dix.Go(initCfgFromJsonDebug(time.Now().Format("2006-01-02 15:04:05")))
		time.Sleep(time.Second)
	}
}

func initCfgFromJsonDebug(name string) xlog.XLog {
	cfg := `{
        "level": "debug",
        "development": true,
        "disableCaller": false,
        "disableStacktrace": false,
        "sampling": null,
        "encoding": "console",
        "encoderConfig": {
                "messageKey": "M",
                "levelKey": "L",
                "timeKey": "T",
                "nameKey": "N",
                "callerKey": "C",
                "stacktraceKey": "S",
                "lineEnding": "\n",
                "levelEncoder": "capitalColor",
                "timeEncoder": "iso8601",
                "durationEncoder": "string",
                "callerEncoder": "default",
                "nameEncoder": ""
        },
        "outputPaths": [
                "stderr"
        ],
        "errorOutputPaths": [
                "stderr"
        ],
        "initialFields": null
}`

	xx, err := xlog_config.NewFromJson(
		[]byte(cfg),
	)
	xerror.Exit(err)
	return xx.Named(name)
}
