package dixinternal

import (
	"strings"
)

// isCycle Check whether type circular dependency
func (x *Dix) isCycle() (string, bool) {
	depGraph := buildDependencyGraph(x.providers)

	cyclePath := detectCycle(depGraph)
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
