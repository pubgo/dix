package main

import (
	"reflect"
	"testing"
)

// 锁定 example/inject-map-list 的契约:同 key 的切片产物按注册顺序拼接,
// 不同 key 独立分组。
func TestBuildRoutes(t *testing.T) {
	routes := buildRoutes()

	if got, want := paths(routes["api"]), []string{"/users", "/orders", "/health"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("api routes = %v, want registration order %v", got, want)
	}
	if got, want := paths(routes["web"]), []string{"/home"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("web routes = %v, want %v", got, want)
	}
}
