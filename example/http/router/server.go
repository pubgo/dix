package router

import (
	"log"
	"net"
	"net/http"
	"os"

	"github.com/pubgo/dix/v2/dixhttp"
)

const defaultHTTPAddr = ":8080"

// StartVisualizationServer 启动 dixhttp 可视化服务。
func StartVisualizationServer(server *dixhttp.Server) error {
	addr := os.Getenv("DIX_HTTP_ADDR")
	if addr == "" {
		addr = defaultHTTPAddr
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		if addr == defaultHTTPAddr {
			log.Printf("⚠️ Port %s unavailable (%v), trying a random available port...", addr, err)
			ln, err = net.Listen("tcp", ":0")
		}
		if err != nil {
			return err
		}
	}

	actualAddr := ln.Addr().String()
	displayAddr := actualAddr
	if _, port, splitErr := net.SplitHostPort(actualAddr); splitErr == nil && port != "" {
		displayAddr = "localhost:" + port
	}

	log.Printf("🚀 Starting HTTP server on http://%s", displayAddr)
	log.Printf("📊 Open http://%s in your browser (legacy DI architecture UI)", displayAddr)
	log.Println("📡 API endpoints:")
	log.Println("   - GET /api/dependencies - JSON data of dependencies")
	log.Println("   - GET /api/modules      - module-level aggregation")
	log.Println("   - GET /api/ego          - neighborhood subgraph")
	log.Println("   - GET /api/search       - server-side graph search")
	log.Println("   - GET /api/stats        - overview statistics")
	log.Println("   - GET /api/runtime-stats - provider startup timings")
	log.Println("   - GET /api/errors       - recent inject errors")
	log.Println("   - GET /api/diagnostics  - DIX_DIAG_FILE records")
	log.Println("   - GET /api/trace        - dixtrace event query")
	log.Println("   - GET /api/trace-tree   - nested call tree per trace")
	return (&http.Server{Handler: server}).Serve(ln)
}
