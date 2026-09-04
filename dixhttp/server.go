package dixhttp

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/pubgo/dix/v2"
	"github.com/pubgo/dix/v2/dixinternal"
	"github.com/pubgo/dix/v2/dixtrace"
)

//go:embed static
var staticFS embed.FS

// Server provides HTTP endpoints to visualize dependency relationships
type Server struct {
	dix *dix.Dix
	mux *http.ServeMux
	// basePath is an optional URL prefix (no trailing slash). Example: "/dix"
	basePath string

	// 依赖数据快照缓存:按 GraphVersion 判脏。反射(ProviderDetails/objects)
	// 与全量投影只在版本变化后的第一次请求发生;未过滤请求返回同一份快照,
	// 过滤请求基于缓存输入重建(零反射)。
	snapMu   sync.Mutex
	snapVer  uint64
	snapDet  []dixinternal.ProviderDetails
	snapObj  map[reflect.Type]map[string][]reflect.Value
	snapFull *DependencyData
	haveSnap bool
}

// GroupRule defines a group name with prefix list for aggregation.
type GroupRule struct {
	Name     string   `json:"name"`
	Prefixes []string `json:"prefixes"`
}

var (
	groupRulesMu sync.RWMutex
	groupRules   []GroupRule
)

// RegisterGroupRules registers global group rules for visualization.
// This can be called by business code to predefine group rules.
func RegisterGroupRules(rules ...GroupRule) {
	groupRulesMu.Lock()
	defer groupRulesMu.Unlock()
	groupRules = sanitizeGroupRules(rules)
}

func getGroupRules() []GroupRule {
	groupRulesMu.RLock()
	defer groupRulesMu.RUnlock()
	if len(groupRules) == 0 {
		return nil
	}
	result := make([]GroupRule, 0, len(groupRules))
	for _, r := range groupRules {
		result = append(result, GroupRule{Name: r.Name, Prefixes: append([]string{}, r.Prefixes...)})
	}
	return result
}

func sanitizeGroupRules(rules []GroupRule) []GroupRule {
	var result []GroupRule
	seen := make(map[string]bool)
	for _, r := range rules {
		name := strings.TrimSpace(r.Name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		var prefixes []string
		prefixSeen := make(map[string]bool)
		for _, p := range r.Prefixes {
			pp := strings.TrimSpace(p)
			if pp == "" || prefixSeen[pp] {
				continue
			}
			prefixSeen[pp] = true
			prefixes = append(prefixes, pp)
		}
		result = append(result, GroupRule{Name: name, Prefixes: prefixes})
	}
	return result
}

// NewServer creates a new HTTP server for dependency visualization
func NewServer(di *dix.Dix) *Server {
	return NewServerWithOptions(di)
}

// ServerOption customizes the HTTP server behavior.
type ServerOption func(*Server)

// WithBasePath sets an optional URL prefix for all routes. Example: "/dix".
func WithBasePath(basePath string) ServerOption {
	return func(s *Server) {
		s.basePath = normalizeBasePath(basePath)
	}
}

// NewServerWithOptions creates a new HTTP server with options.
func NewServerWithOptions(di *dix.Dix, opts ...ServerOption) *Server {
	s := &Server{
		dix:      di,
		mux:      http.NewServeMux(),
		basePath: "",
	}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	s.setupRoutes()
	return s
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	base := s.basePath
	indexPath := "/"
	if base != "" {
		indexPath = base + "/"
		// Redirect /base -> /base/
		s.mux.HandleFunc(base, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == base {
				http.Redirect(w, r, indexPath, http.StatusMovedPermanently)
				return
			}
			http.NotFound(w, r)
		})
	}

	s.mux.HandleFunc(indexPath, s.HandleIndex)
	s.mux.HandleFunc(base+"/api/dependencies", s.HandleDependencies)
	s.mux.HandleFunc(base+"/api/stats", s.HandleStats)
	s.mux.HandleFunc(base+"/api/runtime-stats", s.HandleRuntimeStats)
	s.mux.HandleFunc(base+"/api/errors", s.HandleErrors)
	s.mux.HandleFunc(base+"/api/diagnostics", s.HandleDiagnostics)
	s.mux.HandleFunc(base+"/api/trace", s.HandleTrace)
	s.mux.HandleFunc(base+"/api/trace-tree", s.HandleTraceTree)
	staticRoot, err := fs.Sub(staticFS, "static")
	if err == nil {
		s.mux.Handle(base+"/static/", http.StripPrefix(base+"/static/", http.FileServer(http.FS(staticRoot))))
	}
	s.mux.HandleFunc(base+"/api/search", s.HandleSearch)
	s.mux.HandleFunc(base+"/api/modules", s.HandleModules)
	s.mux.HandleFunc(base+"/api/ego", s.HandleEgo)
	s.mux.HandleFunc(base+"/api/packages", s.HandlePackages)
	s.mux.HandleFunc(base+"/api/package/", s.HandlePackageDetails)
	s.mux.HandleFunc(base+"/api/type/", s.HandleTypeDetails)
	s.mux.HandleFunc(base+"/api/group-rules", s.HandleGroupRules)
}

// HandleErrors returns recent Inject/TryInject error events.
// Query params:
// - limit: optional positive integer to limit returned rows.
func (s *Server) HandleErrors(w http.ResponseWriter, r *http.Request) {
	limit := 0
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	errors := s.dix.GetRecentErrors(limit)
	writeJSON(w, errors)
}

// HandleDiagnostics returns records from DIX_DIAG_FILE (JSONL).
// Query params:
// - kind: trace|error (optional)
// - event: trace event fuzzy match (optional)
// - q: full-text search over record JSON (optional)
// - limit: optional positive integer, default 200, max 2000
// - before_id: optional record id cursor for older-page query
// - since_unix_nano: optional lower time bound
// - until_unix_nano: optional upper time bound
func (s *Server) HandleDiagnostics(w http.ResponseWriter, r *http.Request) {
	query := dixinternal.DiagFileQuery{
		Kind:   strings.TrimSpace(r.URL.Query().Get("kind")),
		Event:  strings.TrimSpace(r.URL.Query().Get("event")),
		Search: strings.TrimSpace(r.URL.Query().Get("q")),
	}

	if limitStr := strings.TrimSpace(r.URL.Query().Get("limit")); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			query.Limit = l
		}
	}

	if beforeStr := strings.TrimSpace(r.URL.Query().Get("before_id")); beforeStr != "" {
		if before, err := strconv.ParseInt(beforeStr, 10, 64); err == nil {
			query.BeforeID = before
		}
	}

	if sinceStr := strings.TrimSpace(r.URL.Query().Get("since_unix_nano")); sinceStr != "" {
		if since, err := strconv.ParseInt(sinceStr, 10, 64); err == nil {
			query.SinceUnix = since
		}
	}

	if untilStr := strings.TrimSpace(r.URL.Query().Get("until_unix_nano")); untilStr != "" {
		if until, err := strconv.ParseInt(untilStr, 10, 64); err == nil {
			query.UntilUnix = until
		}
	}

	result, err := dixinternal.ReadDiagFileRecords(query)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read diagnostics: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, result)
}

// HandleTrace returns in-memory dixtrace events.
// Query params:
// - trace_id, operation, status, event, component, provider, output_type, q
// - limit, before_id, since_unix_nano, until_unix_nano
func (s *Server) HandleTrace(w http.ResponseWriter, r *http.Request) {
	params := map[string]any{
		"trace_id":        strings.TrimSpace(r.URL.Query().Get("trace_id")),
		"container_id":    strings.TrimSpace(r.URL.Query().Get("container_id")),
		"operation":       strings.TrimSpace(r.URL.Query().Get("operation")),
		"status":          strings.TrimSpace(r.URL.Query().Get("status")),
		"event":           strings.TrimSpace(r.URL.Query().Get("event")),
		"component":       strings.TrimSpace(r.URL.Query().Get("component")),
		"provider":        strings.TrimSpace(r.URL.Query().Get("provider")),
		"output_type":     strings.TrimSpace(r.URL.Query().Get("output_type")),
		"q":               strings.TrimSpace(r.URL.Query().Get("q")),
		"limit":           strings.TrimSpace(r.URL.Query().Get("limit")),
		"before_id":       strings.TrimSpace(r.URL.Query().Get("before_id")),
		"since_unix_nano": strings.TrimSpace(r.URL.Query().Get("since_unix_nano")),
		"until_unix_nano": strings.TrimSpace(r.URL.Query().Get("until_unix_nano")),
	}

	result := dixtrace.QueryEvents(dixtrace.ParseQueryFromMap(params))
	writeJSON(w, result)
}

// HandleTraceTree returns the nested call tree of one trace.
// Query params:
// - trace_id: required trace id.
func (s *Server) HandleTraceTree(w http.ResponseWriter, r *http.Request) {
	traceID := strings.TrimSpace(r.URL.Query().Get("trace_id"))
	if traceID == "" {
		http.Error(w, "trace_id required", http.StatusBadRequest)
		return
	}
	writeJSON(w, s.dix.TraceTree(traceID))
}

// HandleSearch 检索图节点。
// Query params:
// - q: 关键字(类型名/函数名包含匹配)
// - kind: type|provider|object
// - module: pkg 前缀
// - state: instantiated|error|slow
// - limit: 缺省 50,上限 500
func (s *Server) HandleSearch(w http.ResponseWriter, r *http.Request) {
	hits := s.dix.SearchNodes(
		r.URL.Query().Get("q"),
		r.URL.Query().Get("kind"),
		r.URL.Query().Get("module"),
		r.URL.Query().Get("state"),
		atoiOr(r.URL.Query().Get("limit"), 50),
	)
	writeJSON(w, hits)
}

// HandleModules 返回模块级聚合视图(含跨模块依赖)。
func (s *Server) HandleModules(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.dix.ModuleGraph())
}

// HandleEgo 返回以 center 为中心的 N 跳邻域子图。
// Query params:
// - center: 类型 label(必填)
// - depth: 缺省 2,上限 10
// - direction: both|deps|dependents,缺省 both
func (s *Server) HandleEgo(w http.ResponseWriter, r *http.Request) {
	center := strings.TrimSpace(r.URL.Query().Get("center"))
	if center == "" {
		http.Error(w, "center required", http.StatusBadRequest)
		return
	}
	writeJSON(w, s.dix.EgoGraph(center, atoiOr(r.URL.Query().Get("depth"), 2), r.URL.Query().Get("direction")))
}

func atoiOr(s string, def int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(s)); err == nil && v > 0 {
		return v
	}
	return def
}

// ServeHTTP implements http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// ListenAndServe starts the HTTP server on the specified address
func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s)
}

// HandleIndex serves the HTML visualization page
func (s *Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	index, err := fs.ReadFile(staticFS, "static/index.html")
	if err != nil {
		http.Error(w, "index not found", http.StatusInternalServerError)
		return
	}
	html := strings.ReplaceAll(string(index), "__DIX_BASE_PATH__", s.basePath)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

// HandleStats returns summary statistics
func (s *Server) HandleStats(w http.ResponseWriter, r *http.Request) {
	providerDetails, objects := s.cachedGraphInputs()
	modules := s.dix.ModuleGraph()

	// Count objects
	objectCount := 0
	for _, groupsMap := range objects {
		for _, values := range groupsMap {
			objectCount += len(values)
		}
	}

	// Count unique packages
	packages := make(map[string]bool)
	for _, detail := range providerDetails {
		pkg := extractPackage(detail.OutputType)
		if pkg != "" {
			packages[pkg] = true
		}
	}

	// Count edges
	edgeCount := 0
	for _, detail := range providerDetails {
		edgeCount += len(detail.InputTypes)
	}

	slow, errored := s.dix.ProblemProviders()
	top := s.dix.ResolvedTopN(10)
	top2 := make([]ResolvedCount2, 0, len(top))
	for _, rc := range top {
		top2 = append(top2, ResolvedCount2{Type: rc.Type, Count: rc.Count})
	}

	stats := StatsData{
		ProviderCount:  len(providerDetails),
		ObjectCount:    objectCount,
		PackageCount:   len(packages),
		EdgeCount:      edgeCount,
		Modules:        len(modules),
		TopResolved:    top2,
		SlowProviders:  slow,
		ErrorProviders: errored,
	}

	writeJSON(w, stats)
}

// HandleRuntimeStats returns provider runtime stats for startup/perf diagnosis.
// Query params:
// - limit: optional positive integer to limit returned rows.
func (s *Server) HandleRuntimeStats(w http.ResponseWriter, r *http.Request) {
	limit := 0
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	stats := s.dix.GetProviderRuntimeStats()
	if limit > 0 && len(stats) > limit {
		stats = stats[:limit]
	}

	writeJSON(w, stats)
}

// HandlePackages returns list of packages for navigation
func (s *Server) HandlePackages(w http.ResponseWriter, r *http.Request) {
	providerDetails, _ := s.cachedGraphInputs()

	// Group by package
	packageMap := make(map[string]*PackageInfo)
	for _, detail := range providerDetails {
		pkg := extractPackage(detail.OutputType)
		if pkg == "" {
			pkg = "(anonymous)"
		}

		if _, exists := packageMap[pkg]; !exists {
			packageMap[pkg] = &PackageInfo{
				Name:          pkg,
				ProviderCount: 0,
				Types:         make([]string, 0),
			}
		}

		packageMap[pkg].ProviderCount++

		// Track unique types
		found := false
		for _, t := range packageMap[pkg].Types {
			if t == detail.OutputType {
				found = true
				break
			}
		}
		if !found {
			packageMap[pkg].Types = append(packageMap[pkg].Types, detail.OutputType)
		}
	}

	// Convert to slice
	packages := make([]PackageInfo, 0, len(packageMap))
	for _, pkg := range packageMap {
		packages = append(packages, *pkg)
	}

	writeJSON(w, packages)
}

// HandlePackageDetails returns details for a specific package
func (s *Server) HandlePackageDetails(w http.ResponseWriter, r *http.Request) {
	// Extract package name from URL
	prefix := s.basePath + "/api/package/"
	pkgName := strings.TrimPrefix(r.URL.Path, prefix)
	if pkgName == r.URL.Path {
		http.Error(w, "package name required", http.StatusBadRequest)
		return
	}
	if pkgName == "" {
		http.Error(w, "package name required", http.StatusBadRequest)
		return
	}

	providerDetails, _ := s.cachedGraphInputs()

	// Filter providers by package
	var providers []ProviderInfo
	for i, detail := range providerDetails {
		pkg := extractPackage(detail.OutputType)
		if pkg == "" {
			pkg = "(anonymous)"
		}
		if pkg != pkgName {
			continue
		}

		providerID := fmt.Sprintf("provider_%s_%d", detail.OutputType, i)
		providers = append(providers, ProviderInfo{
			ID:           providerID,
			OutputType:   detail.OutputType,
			FunctionName: detail.FunctionName,
			InputTypes:   detail.InputTypes,
		})
	}

	// Build edges within package
	var edges []EdgeInfo
	typeSet := make(map[string]bool)
	for _, p := range providers {
		typeSet[p.OutputType] = true
	}

	for _, p := range providers {
		for _, inputType := range p.InputTypes {
			edges = append(edges, EdgeInfo{
				From: inputType,
				To:   p.OutputType,
				Type: "dependency",
			})
		}
	}

	result := PackageDetailsData{
		Package:   pkgName,
		Providers: providers,
		Edges:     edges,
	}

	writeJSON(w, result)
}

// HandleTypeDetails returns dependency details for a specific type
func (s *Server) HandleTypeDetails(w http.ResponseWriter, r *http.Request) {
	// Extract type name from URL
	prefix := s.basePath + "/api/type/"
	typeName := strings.TrimPrefix(r.URL.Path, prefix)
	if typeName == r.URL.Path {
		http.Error(w, "type name required", http.StatusBadRequest)
		return
	}
	if typeName == "" {
		http.Error(w, "type name required", http.StatusBadRequest)
		return
	}

	// Parse depth parameter
	depth := 2
	if depthStr := r.URL.Query().Get("depth"); depthStr != "" {
		if d, err := strconv.Atoi(depthStr); err == nil && d > 0 {
			depth = d
		}
	}

	providerDetails, _ := s.cachedGraphInputs()

	// Build type -> provider mapping
	typeToProvider := make(map[string][]dixinternal.ProviderDetails)
	for _, detail := range providerDetails {
		typeToProvider[detail.OutputType] = append(typeToProvider[detail.OutputType], detail)
	}

	// BFS to find dependencies up to depth
	visited := make(map[string]bool)
	var nodes []TypeNode
	var edges []EdgeInfo

	queue := []struct {
		typeName string
		level    int
	}{{typeName, 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current.typeName] || current.level > depth {
			continue
		}
		visited[current.typeName] = true

		// Add node
		pkg := extractPackage(current.typeName)
		nodes = append(nodes, TypeNode{
			ID:      current.typeName,
			Type:    current.typeName,
			Package: pkg,
			Level:   current.level,
		})

		// Find providers for this type
		providers := typeToProvider[current.typeName]
		for _, p := range providers {
			for _, inputType := range p.InputTypes {
				// Add edge
				edges = append(edges, EdgeInfo{
					From: inputType,
					To:   current.typeName,
					Type: "dependency",
				})

				// Queue input type for processing
				if !visited[inputType] {
					queue = append(queue, struct {
						typeName string
						level    int
					}{inputType, current.level + 1})
				}
			}
		}
	}

	result := TypeDetailsData{
		RootType: typeName,
		Depth:    depth,
		Nodes:    nodes,
		Edges:    edges,
	}

	writeJSON(w, result)
}

// HandleDependencies returns JSON data about providers and objects relationships
// cachedGraphInputs 返回按图版本判脏缓存的 provider 详情与对象表。
func (s *Server) cachedGraphInputs() ([]dixinternal.ProviderDetails, map[reflect.Type]map[string][]reflect.Value) {
	ver := s.dix.GraphVersion()
	s.snapMu.Lock()
	defer s.snapMu.Unlock()
	if !s.haveSnap || s.snapVer != ver {
		s.snapDet = s.dix.GetProviderDetails()
		s.snapObj = s.dix.GetObjects()
		s.snapVer = ver
		s.haveSnap = true
	}
	return s.snapDet, s.snapObj
}

func (s *Server) HandleDependencies(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for filtering
	pkgFilter := r.URL.Query().Get("package")
	limitStr := r.URL.Query().Get("limit")
	limit := 0 // No limit by default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	data := s.dependencyData(pkgFilter, limit)

	writeJSON(w, data)
}

// HandleGroupRules returns global group rules for visualization.
func (s *Server) HandleGroupRules(w http.ResponseWriter, r *http.Request) {
	rules := getGroupRules()
	if rules == nil {
		rules = []GroupRule{}
	}
	writeJSON(w, rules)
}

// dependencyData 组装依赖数据:未过滤请求返回缓存的全量快照(零反射、零分配),
// 过滤请求基于缓存输入做纯函数投影;反射只在图版本变化后的第一次请求发生。
func (s *Server) dependencyData(pkgFilter string, limit int) *DependencyData {
	ver := s.dix.GraphVersion()

	s.snapMu.Lock()
	if !s.haveSnap || s.snapVer != ver {
		s.snapDet = s.dix.GetProviderDetails()
		s.snapObj = s.dix.GetObjects()
		s.snapVer = ver
		s.haveSnap = true
		s.snapFull = buildDependencyData(s.snapDet, s.snapObj, "", 0)
	}
	if pkgFilter == "" && limit == 0 {
		full := s.snapFull
		s.snapMu.Unlock()
		return full
	}
	details, objects := s.snapDet, s.snapObj
	s.snapMu.Unlock()
	return buildDependencyData(details, objects, pkgFilter, limit)
}

// buildDependencyData 从 provider 详情与对象表提取结构化依赖数据(纯函数)。
func buildDependencyData(details []dixinternal.ProviderDetails, objects map[reflect.Type]map[string][]reflect.Value, pkgFilter string, limit int) *DependencyData {
	data := &DependencyData{
		Providers: []ProviderInfo{},
		Objects:   []ObjectInfo{},
		Edges:     []EdgeInfo{},
	}

	data.Providers = aggregateProviderInfos(details, pkgFilter, limit)

	for _, provider := range data.Providers {
		outputTypes := provider.OutputTypes
		if len(outputTypes) == 0 && provider.OutputType != "" {
			outputTypes = []string{provider.OutputType}
		}
		for _, outputType := range outputTypes {
			for _, inputTypeStr := range provider.InputTypes {
				data.Edges = append(data.Edges, EdgeInfo{
					From: inputTypeStr,
					To:   outputType,
					Type: "provider",
				})
			}
		}
	}

	// Extract object information using the cached objects table
	for outputType, groupsMap := range objects {
		// Apply package filter if specified
		if pkgFilter != "" {
			pkg := extractPackage(outputType.String())
			if pkg != pkgFilter {
				continue
			}
		}

		for group, values := range groupsMap {
			for i, value := range values {
				objectID := fmt.Sprintf("object_%s_%s_%d", outputType.String(), group, i)

				objectInfo := ObjectInfo{
					ID:            objectID,
					Type:          outputType.String(),
					Group:         group,
					IsInitialized: value.IsValid() && !value.IsZero(),
				}

				data.Objects = append(data.Objects, objectInfo)

				// Add edge from provider output to object
				data.Edges = append(data.Edges, EdgeInfo{
					From: outputType.String(),
					To:   objectID,
					Type: "object",
				})
			}
		}
	}

	return data
}

func aggregateProviderInfos(details []dixinternal.ProviderDetails, pkgFilter string, limit int) []ProviderInfo {
	type providerBucket struct {
		provider   ProviderInfo
		outputSeen map[string]bool
		inputSeen  map[string]bool
	}

	buckets := make(map[string]*providerBucket)
	order := make([]string, 0, len(details))

	for _, detail := range details {
		if pkgFilter != "" {
			pkg := extractPackage(detail.OutputType)
			if pkg != pkgFilter {
				continue
			}
		}

		key := providerAggregateKey(detail)
		bucket, exists := buckets[key]
		if !exists {
			bucket = &providerBucket{
				provider: ProviderInfo{
					ID:           "provider_" + key,
					OutputType:   detail.OutputType,
					OutputPkg:    detail.OutputPkg,
					FunctionName: detail.FunctionName,
					FunctionPkg:  detail.FunctionPkg,
					FunctionFile: detail.FunctionFile,
					FunctionLine: detail.FunctionLine,
					OutputTypes:  make([]string, 0, 4),
					InputTypes:   make([]string, 0, 8),
					InputPkgs:    make([]string, 0, 8),
				},
				outputSeen: make(map[string]bool),
				inputSeen:  make(map[string]bool),
			}
			buckets[key] = bucket
			order = append(order, key)
		}

		if out := strings.TrimSpace(detail.OutputType); out != "" && !bucket.outputSeen[out] {
			bucket.outputSeen[out] = true
			bucket.provider.OutputTypes = append(bucket.provider.OutputTypes, out)
			if bucket.provider.OutputType == "" {
				bucket.provider.OutputType = out
			}
		}

		for i, in := range detail.InputTypes {
			in = strings.TrimSpace(in)
			if in == "" || bucket.inputSeen[in] {
				continue
			}
			bucket.inputSeen[in] = true
			bucket.provider.InputTypes = append(bucket.provider.InputTypes, in)

			pkg := ""
			if i < len(detail.InputPkgs) {
				pkg = strings.TrimSpace(detail.InputPkgs[i])
			}
			bucket.provider.InputPkgs = append(bucket.provider.InputPkgs, pkg)
		}
	}

	providers := make([]ProviderInfo, 0, len(order))
	for _, key := range order {
		providers = append(providers, buckets[key].provider)
	}

	if limit > 0 && len(providers) > limit {
		providers = providers[:limit]
	}

	return providers
}

func providerAggregateKey(detail dixinternal.ProviderDetails) string {
	if detail.FunctionFile != "" && detail.FunctionLine > 0 {
		return fmt.Sprintf("%s:%d", detail.FunctionFile, detail.FunctionLine)
	}
	if detail.FunctionName != "" {
		return detail.FunctionName
	}
	if detail.OutputType != "" {
		return detail.OutputType
	}
	return "unknown"
}

// Data types

// StatsData contains summary statistics
type StatsData struct {
	ProviderCount int `json:"provider_count"`
	ObjectCount   int `json:"object_count"`
	PackageCount  int `json:"package_count"`
	EdgeCount     int `json:"edge_count"`

	// 概览增强(P4a):模块数、解析热度 TopN、慢/错误 provider
	Modules        int              `json:"modules"`
	TopResolved    []ResolvedCount2 `json:"top_resolved,omitempty"`
	SlowProviders  []string         `json:"slow_providers,omitempty"`
	ErrorProviders []string         `json:"error_providers,omitempty"`
}

// ResolvedCount2 是 dixhttp 对内部分析计数类型的投影。
type ResolvedCount2 struct {
	Type  string `json:"type"`
	Count int64  `json:"count"`
}

// PackageInfo contains information about a package
type PackageInfo struct {
	Name          string   `json:"name"`
	ProviderCount int      `json:"provider_count"`
	Types         []string `json:"types"`
}

// PackageDetailsData contains detailed information for a package
type PackageDetailsData struct {
	Package   string         `json:"package"`
	Providers []ProviderInfo `json:"providers"`
	Edges     []EdgeInfo     `json:"edges"`
}

// TypeNode represents a type in the dependency graph
type TypeNode struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Package string `json:"package"`
	Level   int    `json:"level"`
}

// TypeDetailsData contains dependency details for a type
type TypeDetailsData struct {
	RootType string     `json:"root_type"`
	Depth    int        `json:"depth"`
	Nodes    []TypeNode `json:"nodes"`
	Edges    []EdgeInfo `json:"edges"`
}

// DependencyData represents the structure of dependency information
type DependencyData struct {
	Providers []ProviderInfo `json:"providers"`
	Objects   []ObjectInfo   `json:"objects"`
	Edges     []EdgeInfo     `json:"edges"`
}

// ProviderInfo contains information about a provider
type ProviderInfo struct {
	ID           string   `json:"id"`
	OutputType   string   `json:"output_type"`
	OutputTypes  []string `json:"output_types,omitempty"`
	OutputPkg    string   `json:"output_pkg"`
	FunctionName string   `json:"function_name"`
	FunctionPkg  string   `json:"function_pkg"`
	FunctionFile string   `json:"function_file"`
	FunctionLine int      `json:"function_line"`
	InputTypes   []string `json:"input_types"`
	InputPkgs    []string `json:"input_pkgs"`
}

// ObjectInfo contains information about an object instance
type ObjectInfo struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	Group         string `json:"group"`
	IsInitialized bool   `json:"is_initialized"`
}

// EdgeInfo represents a dependency relationship
type EdgeInfo struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"` // "provider", "object", "dependency"
}

// Helper functions

func extractPackage(typeName string) string {
	// Handle pointer types
	typeName = strings.TrimPrefix(typeName, "*")

	// Handle slice/array types
	typeName = strings.TrimPrefix(typeName, "[]")

	// Handle map types - extract value type package
	if strings.HasPrefix(typeName, "map[") {
		// Find the value type after ]
		idx := strings.Index(typeName, "]")
		if idx > 0 && idx < len(typeName)-1 {
			typeName = typeName[idx+1:]
			typeName = strings.TrimPrefix(typeName, "*")
		}
	}

	// Find the last dot before the type name
	lastSlash := strings.LastIndex(typeName, "/")
	if lastSlash == -1 {
		// No slash, check for simple package.Type format
		dotIdx := strings.LastIndex(typeName, ".")
		if dotIdx > 0 {
			return typeName[:dotIdx]
		}
		return ""
	}

	// Find the dot after the last slash (package.Type)
	afterSlash := typeName[lastSlash+1:]
	dotIdx := strings.Index(afterSlash, ".")
	if dotIdx > 0 {
		return typeName[:lastSlash+1+dotIdx]
	}

	return typeName
}

func writeJSON(w http.ResponseWriter, data any) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode JSON: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

func normalizeBasePath(basePath string) string {
	base := strings.TrimSpace(basePath)
	if base == "" || base == "/" {
		return ""
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	base = strings.TrimRight(base, "/")
	return base
}
