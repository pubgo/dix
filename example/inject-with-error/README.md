# Inject 函数支持 Error 返回值

这个示例演示了依赖注入框架的一个重要功能：**Inject 函数现在支持有 error 返回值的函数**。

## 功能说明

从版本 X.X.X 开始，`Inject` 方法支持以下两种函数签名：

### 1. 无返回值函数（原有功能）
```go
func myFunction(dep1 Dependency1, dep2 Dependency2) {
    // 处理逻辑
}
```

### 2. 有 error 返回值函数（新功能）
```go
func myFunction(dep1 Dependency1, dep2 Dependency2) error {
    // 处理逻辑，可能返回错误
    return someOperation()
}
```

## 使用场景

### 初始化操作
```go
func initDatabase(logger Logger, db Database) error {
    if err := db.Connect(); err != nil {
        return fmt.Errorf("数据库连接失败: %w", err)
    }
    return db.Initialize()
}

// 使用
err := container.Inject(initDatabase)
if err != nil {
    log.Fatal("初始化失败:", err)
}
```

### 启动服务
```go
func startServer(logger Logger, server *HTTPServer) error {
    logger.Log("启动服务器...")
    return server.ListenAndServe()
}
```

### 批处理操作
```go
func processBatch(logger Logger, processor *BatchProcessor) error {
    logger.Log("开始批处理...")
    return processor.ProcessAll()
}
```

## 错误处理

- ✅ 如果注入的函数返回 `nil`，注入成功
- ✅ 如果注入的函数返回非 `nil` 错误，该错误会被包装并向上传播
- ✅ 如果函数 panic，panic 会被捕获并转换为错误
- ❌ 如果函数有非 `error` 类型的返回值，注册时会被拒绝

## 限制

1. **只支持一个返回值**：函数最多只能有一个返回值，且必须是 `error` 类型
2. **参数规则不变**：函数参数的类型限制与之前相同
3. **不支持其他返回类型**：不能返回 `string`、`int` 等其他类型

## 示例

运行这个示例：
```bash
go run example/inject-with-error/main.go
```

你会看到以下测试：
- ✅ 无返回值函数注入
- ✅ 有 error 返回值且成功的函数注入  
- ✅ 有 error 返回值且失败的函数注入（错误被正确捕获）
- ✅ 拒绝非 error 返回值的函数
- ✅ 匿名函数支持
- ✅ dixglobal.Inject 的便捷用法

## 向后兼容性

此功能完全向后兼容。现有的无返回值函数注入继续正常工作，没有任何破坏性变更。 