package dixhttp

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pubgo/dix/v2/dixinternal"
)

//go:embed template.html
var htmlTemplate string

// Server provides HTTP endpoints to visualize dependency relationships
type Server struct {
	dix *dixinternal.Dix
	mux *http.ServeMux
}

// NewServer creates a new HTTP server for dependency visualization
func NewServer(dix *dixinternal.Dix) *Server {
	s := &Server{
		dix: dix,
		mux: http.NewServeMux(),
	}
	s.setupRoutes()
	return s
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.HandleFunc("/api/dependencies", s.handleDependencies)
	s.mux.HandleFunc("/api/graph", s.handleGraph)
}

// ServeHTTP implements http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// ListenAndServe starts the HTTP server on the specified address
func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s)
}

// handleIndex serves the HTML visualization page
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, htmlTemplate)
}

// handleDependencies returns JSON data about providers and objects relationships
func (s *Server) handleDependencies(w http.ResponseWriter, r *http.Request) {
	data := s.extractDependencyData()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode JSON: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleGraph returns the DOT graph format
func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	graphType := r.URL.Query().Get("type")
	if graphType == "" {
		graphType = "providers"
	}

	var dotContent string
	graph := s.dix.Graph()
	switch graphType {
	case "providers":
		dotContent = graph.Providers
	case "provider_types":
		dotContent = graph.ProviderTypes
	case "objects":
		dotContent = graph.Objects
	default:
		http.Error(w, "Invalid graph type. Use: providers, provider_types, or objects", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, dotContent)
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
	FunctionName string   `json:"function_name"`
	InputTypes   []string `json:"input_types"`
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
	Type string `json:"type"` // "provider" or "object"
}

// extractDependencyData extracts structured data from the Dix container
func (s *Server) extractDependencyData() *DependencyData {
	data := &DependencyData{
		Providers: []ProviderInfo{},
		Objects:   []ObjectInfo{},
		Edges:     []EdgeInfo{},
	}

	// Extract provider information using the public API
	providerDetails := s.dix.GetProviderDetails()

	for i, detail := range providerDetails {
		providerID := fmt.Sprintf("provider_%s_%d", detail.OutputType, i)
		providerInfo := ProviderInfo{
			ID:           providerID,
			OutputType:   detail.OutputType,
			FunctionName: detail.FunctionName,
			InputTypes:   detail.InputTypes,
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
