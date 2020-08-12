package main

import (
	"encoding/json"
	"fmt"
	"github.com/pubgo/dix"
	"github.com/pubgo/xerror"
	"log"
	"os"
	"time"
)

type Config struct {
	Prefix string
}

func main() {

	defer xerror.RespDebug()

	i := 0
	var di = dix.New()
	err := di.Dix(func(cfg *Config) (*log.Logger, error) {
		return log.New(os.Stdout, cfg.Prefix, log.Llongfile), nil
	})
	if err != nil {
		panic(err)
	}
	err = di.Dix(func(l *log.Logger) {
		l.Print("You've been invoked")
	})
	if err != nil {
		panic(err)
	}

	for {
		var cfg Config
		err := json.Unmarshal([]byte(fmt.Sprintf(`{"prefix": "[foo%d] "}`, i)), &cfg)
		xerror.Panic(err)
		xerror.Panic(di.Dix(&cfg))
		fmt.Println(di.Graph())
		time.Sleep(time.Second)
		i++
	}
}
