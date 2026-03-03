package dixhttp

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/pubgo/dix/v2/dixinternal"
)

//go:embed template.html
var htmlTemplate string

// Server provides HTTP endpoints to visualize dependency relationships
type Server struct {
	dix *dixinternal.Dix
	mux *http.ServeMux
	// basePath is an optional URL prefix (no trailing slash). Example: "/dix"
	basePath string
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
func NewServer(dix *dixinternal.Dix) *Server {
	return NewServerWithOptions(dix)
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
func NewServerWithOptions(dix *dixinternal.Dix, opts ...ServerOption) *Server {
	s := &Server{
		dix:      dix,
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
	s.mux.HandleFunc(base+"/api/packages", s.HandlePackages)
	s.mux.HandleFunc(base+"/api/package/", s.HandlePackageDetails)
	s.mux.HandleFunc(base+"/api/type/", s.HandleTypeDetails)
	s.mux.HandleFunc(base+"/api/group-rules", s.HandleGroupRules)
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
	html := strings.ReplaceAll(htmlTemplate, "__DIX_BASE_PATH__", s.basePath)
	fmt.Fprint(w, html)
}

// HandleStats returns summary statistics
func (s *Server) HandleStats(w http.ResponseWriter, r *http.Request) {
	providerDetails := s.dix.GetProviderDetails()
	objects := s.dix.GetObjects()

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

	stats := StatsData{
		ProviderCount: len(providerDetails),
		ObjectCount:   objectCount,
		PackageCount:  len(packages),
		EdgeCount:     edgeCount,
	}

	writeJSON(w, stats)
}

// HandlePackages returns list of packages for navigation
func (s *Server) HandlePackages(w http.ResponseWriter, r *http.Request) {
	providerDetails := s.dix.GetProviderDetails()

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
	pkgName := strings.TrimPrefix(r.URL.Path, "/api/package/")
	if pkgName == "" {
		http.Error(w, "package name required", http.StatusBadRequest)
		return
	}

	providerDetails := s.dix.GetProviderDetails()

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
	typeName := strings.TrimPrefix(r.URL.Path, "/api/type/")
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

	providerDetails := s.dix.GetProviderDetails()

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

	data := s.extractDependencyData(pkgFilter, limit)

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

// extractDependencyData extracts structured data from the Dix container
func (s *Server) extractDependencyData(pkgFilter string, limit int) *DependencyData {
	data := &DependencyData{
		Providers: []ProviderInfo{},
		Objects:   []ObjectInfo{},
		Edges:     []EdgeInfo{},
	}

	// Extract provider information using the public API
	providerDetails := s.dix.GetProviderDetails()

	count := 0
	for i, detail := range providerDetails {
		// Apply package filter if specified
		if pkgFilter != "" {
			pkg := extractPackage(detail.OutputType)
			if pkg != pkgFilter {
				continue
			}
		}

		// Apply limit
		if limit > 0 && count >= limit {
			break
		}
		count++

		providerID := fmt.Sprintf("provider_%s_%d", detail.OutputType, i)
		providerInfo := ProviderInfo{
			ID:           providerID,
			OutputType:   detail.OutputType,
			OutputPkg:    detail.OutputPkg,
			FunctionName: detail.FunctionName,
			FunctionPkg:  detail.FunctionPkg,
			InputTypes:   detail.InputTypes,
			InputPkgs:    detail.InputPkgs,
		}

		// Add edges from input types to provider output
		for _, inputTypeStr := range detail.InputTypes {
			data.Edges = append(data.Edges, EdgeInfo{
				From: inputTypeStr,
				To:   detail.OutputType,
				Type: "provider",
			})
		}

		data.Providers = append(data.Providers, providerInfo)
	}

	// Extract object information using the public API
	objects := s.dix.GetObjects()

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

// Data types

// StatsData contains summary statistics
type StatsData struct {
	ProviderCount int `json:"provider_count"`
	ObjectCount   int `json:"object_count"`
	PackageCount  int `json:"package_count"`
	EdgeCount     int `json:"edge_count"`
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
	OutputPkg    string   `json:"output_pkg"`
	FunctionName string   `json:"function_name"`
	FunctionPkg  string   `json:"function_pkg"`
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
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode JSON: %v", err), http.StatusInternalServerError)
	}
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
