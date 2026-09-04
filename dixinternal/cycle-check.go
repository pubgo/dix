package dixinternal

import (
	"strings"
)

// isCycle 检查依赖环。依赖图由 Graph 在 Provide 时增量维护,
// 这里只做只读投影 + DFS,不再有全量重建步骤。
// 确定性语义(#57):起点与邻居按类型名字典序,报告化简后的环路径。
func (dix *Dix) isCycle() (string, bool) {
	cyclePath := detectCycle(dix.graph.declaredAdjacency())
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
