package dix_envs

import (
	"os"
	"strconv"
	"strings"
)

const (
	Trace = "dix_trace"
)

func IsTrace() bool {
	b, _ := strconv.ParseBool(os.Getenv(strings.ToUpper(Trace)))
	return b
}

func SetTrace() {
	_ = os.Setenv(strings.ToUpper(Trace), "true")
}
