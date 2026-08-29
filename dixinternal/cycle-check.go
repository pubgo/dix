package dixinternal

import (
	"strings"
)

// isCycle Check whether type circular dependency.
// The dependency graph is cached and rebuilt only after new providers are
// registered, instead of on every Inject/TryInject call.
func (dix *Dix) isCycle() (string, bool) {
	if dix.graphDirty {
		dix.depGraph = buildDependencyGraph(dix.providers)
		dix.graphDirty = false
	}

	cyclePath := detectCycle(dix.depGraph)
	if len(cyclePath) == 0 {
		return "", false
	}

	var pathStr strings.Builder
	for i, t := range cyclePath {
		if i > 0 {
			pathStr.WriteString(" -> ")
		}
		pathStr.WriteString(t.String())
	}
	return pathStr.String(), true
}
