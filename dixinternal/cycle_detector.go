package dixinternal

import (
	"reflect"
	"slices"
	"strings"
)

// CycleDetectorImpl 循环依赖检测器实现
type CycleDetectorImpl struct{}

// NewCycleDetector 创建新的循环依赖检测器
func NewCycleDetector() *CycleDetectorImpl {
	return &CycleDetectorImpl{}
}

// DetectCycle 检测循环依赖
func (cd *CycleDetectorImpl) DetectCycle(providers map[reflect.Type][]Provider) ([]reflect.Type, error) {
	graph := cd.buildDependencyGraph(providers)
	return cd.detectCycleInGraph(graph), nil
}

// buildDependencyGraph 构建依赖关系图
func (cd *CycleDetectorImpl) buildDependencyGraph(providers map[reflect.Type][]Provider) map[reflect.Type]map[reflect.Type]bool {
	graph := make(map[reflect.Type]map[reflect.Type]bool)

	// 预分配map容量以减少rehash
	for outputType := range providers {
		graph[outputType] = make(map[reflect.Type]bool)
	}

	// 构建依赖关系图
	for outputType, providerList := range providers {
		for _, provider := range providerList {
			for _, dep := range provider.Dependencies() {
				// 递归获取所有依赖类型
				depTypes := cd.getAllDependencyTypes(dep)
				for _, depType := range depTypes {
					graph[outputType][depType] = true
				}
			}
		}
	}

	return graph
}

// getAllDependencyTypes 获取依赖的所有类型
func (cd *CycleDetectorImpl) getAllDependencyTypes(dep Dependency) []reflect.Type {
	var types []reflect.Type

	switch dep.Type().Kind() {
	case reflect.Interface, reflect.Ptr, reflect.Func:
		types = append(types, dep.Type())

	case reflect.Struct:
		// 结构体类型，需要递归获取字段依赖
		types = append(types, dep.Type())
		for i := 0; i < dep.Type().NumField(); i++ {
			fieldType := dep.Type().Field(i).Type
			fieldDep := NewDependency(fieldType, false, false)
			types = append(types, cd.getAllDependencyTypes(fieldDep)...)
		}

	case reflect.Map:
		elemType := dep.Type().Elem()
		if elemType.Kind() == reflect.Slice {
			elemType = elemType.Elem()
		}
		elemDep := NewDependency(elemType, false, false)
		types = append(types, cd.getAllDependencyTypes(elemDep)...)

	case reflect.Slice:
		elemType := dep.Type().Elem()
		elemDep := NewDependency(elemType, false, false)
		types = append(types, cd.getAllDependencyTypes(elemDep)...)

	default:
		// 不支持的类型，记录警告但不阻止检测
		logger.Warn().
			Str("type", dep.Type().String()).
			Str("kind", dep.Type().Kind().String()).
			Msg("unsupported dependency type in cycle detection")
	}

	return types
}

// detectCycleInGraph 在依赖图中检测循环
func (cd *CycleDetectorImpl) detectCycleInGraph(graph map[reflect.Type]map[reflect.Type]bool) []reflect.Type {
	visited := make(map[reflect.Type]bool)

	// 深度优先搜索检测循环
	var dfs func(reflect.Type, map[reflect.Type]bool, []reflect.Type) []reflect.Type
	dfs = func(currentType reflect.Type, recursionStack map[reflect.Type]bool, path []reflect.Type) []reflect.Type {
		// 如果当前类型已在递归栈中，说明发现循环
		if recursionStack[currentType] {
			return slices.Clone(path)
		}

		// 如果已访问过，跳过
		if visited[currentType] {
			return nil
		}

		// 标记为已访问和在递归栈中
		visited[currentType] = true
		recursionStack[currentType] = true
		defer delete(recursionStack, currentType)

		// 遍历所有依赖
		for dependencyType := range graph[currentType] {
			cycle := dfs(dependencyType, recursionStack, append(slices.Clone(path), dependencyType))
			if len(cycle) > 0 {
				return cycle
			}
		}

		return nil
	}

	// 对每个类型进行DFS
	for typ := range graph {
		if visited[typ] {
			continue
		}

		cycle := dfs(typ, make(map[reflect.Type]bool), []reflect.Type{typ})
		if len(cycle) > 0 {
			return cycle
		}
	}

	return nil
}

// HasCycle 检查是否存在循环依赖（便捷方法）
func (cd *CycleDetectorImpl) HasCycle(providers map[reflect.Type][]Provider) bool {
	cycle, _ := cd.DetectCycle(providers)
	return len(cycle) > 0
}

// GetCyclePath 获取循环路径的字符串表示
func (cd *CycleDetectorImpl) GetCyclePath(cycle []reflect.Type) string {
	if len(cycle) == 0 {
		return ""
	}

	var pathParts []string
	for _, typ := range cycle {
		pathParts = append(pathParts, typ.String())
	}

	// 添加回到起点的箭头以显示完整循环
	if len(pathParts) > 0 {
		pathParts = append(pathParts, pathParts[0])
	}

	return strings.Join(pathParts, " -> ")
}

// ValidateNoCycles 验证没有循环依赖，如果有则返回错误
func (cd *CycleDetectorImpl) ValidateNoCycles(providers map[reflect.Type][]Provider) error {
	cycle, err := cd.DetectCycle(providers)
	if err != nil {
		return WrapError(err, ErrorTypeCyclicDep, "failed to detect cycles")
	}

	if len(cycle) > 0 {
		return NewCyclicDependencyError(cycle)
	}

	return nil
}
