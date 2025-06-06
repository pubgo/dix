# 贡献指南

感谢您对 Dix 项目的关注！我们欢迎各种形式的贡献，包括但不限于：

- 🐛 报告 Bug
- 💡 提出新功能建议
- 📝 改进文档
- 🔧 提交代码修复
- ✨ 添加新功能
- 🧪 编写测试用例

## 🚀 快速开始

### 环境要求

- **Go 版本**：>= 1.18
- **Git**：用于版本控制
- **Make**：用于构建脚本（可选）

### 设置开发环境

```bash
# 1. Fork 项目到你的 GitHub 账户

# 2. 克隆你的 fork
git clone https://github.com/YOUR_USERNAME/dix.git
cd dix

# 3. 添加上游仓库
git remote add upstream https://github.com/pubgo/dix.git

# 4. 安装依赖
go mod tidy

# 5. 运行测试确保环境正常
go test ./...

# 6. 运行示例
go run example/basic/main.go
```

## 📋 开发流程

### 1. 创建分支

```bash
# 从 main 分支创建新的功能分支
git checkout main
git pull upstream main
git checkout -b feature/your-feature-name

# 或者修复 bug
git checkout -b fix/issue-number-description
```

### 2. 开发和测试

```bash
# 进行开发...

# 运行测试
go test ./...

# 运行特定包的测试
go test ./dixinternal

# 运行带覆盖率的测试
go test -cover ./...

# 运行基准测试
go test -bench=. ./...
```

### 3. 提交代码

```bash
# 添加文件
git add .

# 提交（遵循提交信息规范）
git commit -m "feat: add new dependency injection feature"

# 推送到你的 fork
git push origin feature/your-feature-name
```

### 4. 创建 Pull Request

1. 在 GitHub 上打开你的 fork
2. 点击 "New Pull Request"
3. 选择 `pubgo/dix:main` 作为目标分支
4. 填写 PR 描述（使用模板）
5. 提交 PR

## 📝 代码规范

### Go 代码风格

我们遵循标准的 Go 代码规范：

```bash
# 格式化代码
go fmt ./...

# 检查代码质量
go vet ./...

# 使用 golangci-lint（推荐）
golangci-lint run
```

### 命名规范

- **包名**：小写，简短，有意义
- **函数名**：驼峰命名，公开函数首字母大写
- **变量名**：驼峰命名，简洁明了
- **常量名**：全大写，下划线分隔

```go
// ✅ 好的命名
package dixinternal

type Container interface {
    Provide(provider any) error
    Inject(target any) error
}

func NewContainer() Container {
    return &containerImpl{}
}

const (
    DefaultMaxDepth = 100
    ErrorTypeNotFound = "type not found"
)

// ❌ 不好的命名
package di

type C interface {
    P(p any) error
    I(t any) error
}

func New() C {
    return &cImpl{}
}
```

### 注释规范

- 所有公开的函数、类型、常量都必须有注释
- 注释应该解释"为什么"而不仅仅是"是什么"
- 使用完整的句子，以被注释的标识符开头

```go
// Container 定义了依赖注入容器的核心接口。
// 它负责管理提供者的注册和依赖的解析。
type Container interface {
    // Provide 注册一个提供者函数到容器中。
    // 提供者函数的参数将被自动注入，返回值将被注册为可注入的依赖。
    Provide(provider any) error
    
    // Inject 将依赖注入到目标对象中。
    // 目标可以是结构体指针或函数。
    Inject(target any) error
}
```

## 🧪 测试指南

### 测试结构

```
tests/
├── unit/           # 单元测试
├── integration/    # 集成测试
└── benchmark/      # 性能测试
```

### 编写测试

```go
func TestContainer_Provide(t *testing.T) {
    tests := []struct {
        name    string
        setup   func() Container
        provider any
        wantErr bool
    }{
        {
            name: "valid provider function",
            setup: func() Container {
                return New()
            },
            provider: func() string {
                return "test"
            },
            wantErr: false,
        },
        {
            name: "invalid provider",
            setup: func() Container {
                return New()
            },
            provider: "not a function",
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            container := tt.setup()
            err := container.Provide(tt.provider)
            
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### 基准测试

```go
func BenchmarkContainer_Inject(b *testing.B) {
    container := New()
    container.Provide(func() string { return "test" })
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        var target struct {
            Value string `dix:""`
        }
        container.Inject(&target)
    }
}
```

### 测试覆盖率

我们要求新代码的测试覆盖率至少达到 80%：

```bash
# 生成覆盖率报告
go test -coverprofile=coverage.out ./...

# 查看覆盖率
go tool cover -html=coverage.out

# 检查覆盖率百分比
go tool cover -func=coverage.out
```

## 📚 文档贡献

### 文档类型

- **API 文档**：`docs/API.md`
- **架构文档**：`docs/ARCHITECTURE.md`
- **迁移指南**：`docs/MIGRATION.md`
- **更新日志**：`docs/CHANGELOG.md`
- **示例代码**：`example/` 目录

### 文档规范

- 使用 Markdown 格式
- 包含代码示例
- 保持简洁明了
- 及时更新

### 示例代码

新增示例时请遵循以下结构：

```
example/your-example/
├── main.go          # 主要示例代码
├── README.md        # 示例说明
└── go.mod          # 如果需要特殊依赖
```

## 🐛 Bug 报告

### 报告模板

```markdown
## Bug 描述
简洁明了地描述遇到的问题。

## 复现步骤
1. 执行 '...'
2. 点击 '....'
3. 滚动到 '....'
4. 看到错误

## 期望行为
描述你期望发生的行为。

## 实际行为
描述实际发生的行为。

## 环境信息
- OS: [e.g. macOS 12.0]
- Go 版本: [e.g. 1.19]
- Dix 版本: [e.g. v2.0.0]

## 附加信息
添加任何其他有助于解决问题的信息。
```

## 💡 功能请求

### 请求模板

```markdown
## 功能描述
简洁明了地描述你想要的功能。

## 问题背景
描述这个功能要解决的问题。

## 解决方案
描述你希望的解决方案。

## 替代方案
描述你考虑过的其他解决方案。

## 附加信息
添加任何其他相关信息或截图。
```

## 📋 提交信息规范

我们使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

### 格式

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### 类型

- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档更新
- `style`: 代码格式化
- `refactor`: 代码重构
- `test`: 测试相关
- `chore`: 构建过程或辅助工具的变动

### 示例

```bash
# 新功能
git commit -m "feat(container): add generic Get method"

# Bug 修复
git commit -m "fix(injector): resolve circular dependency detection"

# 文档更新
git commit -m "docs: update API documentation"

# 重构
git commit -m "refactor(provider): simplify provider registration"
```

## 🔍 代码审查

### 审查清单

- [ ] 代码遵循项目规范
- [ ] 包含适当的测试
- [ ] 文档已更新
- [ ] 没有引入破坏性变更
- [ ] 性能影响可接受
- [ ] 安全性考虑充分

### 审查流程

1. **自动检查**：CI/CD 流水线自动运行测试
2. **代码审查**：至少一个维护者审查代码
3. **测试验证**：确保所有测试通过
4. **文档检查**：确保文档完整准确
5. **合并**：审查通过后合并到主分支

## 🏷️ 发布流程

### 版本号规范

我们遵循 [Semantic Versioning](https://semver.org/)：

- `MAJOR.MINOR.PATCH`
- `MAJOR`: 不兼容的 API 变更
- `MINOR`: 向后兼容的功能新增
- `PATCH`: 向后兼容的问题修正

### 发布步骤

1. 更新 `CHANGELOG.md`
2. 创建版本标签
3. 发布 GitHub Release
4. 更新文档

## 🤝 社区

### 沟通渠道

- **GitHub Issues**: 报告 Bug 和功能请求
- **GitHub Discussions**: 一般讨论和问答
- **Pull Requests**: 代码贡献

### 行为准则

我们致力于为每个人提供友好、安全和欢迎的环境：

- 使用友好和包容的语言
- 尊重不同的观点和经验
- 优雅地接受建设性批评
- 关注对社区最有利的事情
- 对其他社区成员表示同理心

## 📞 联系我们

如果你有任何问题或需要帮助，请通过以下方式联系我们：

- 创建 [GitHub Issue](https://github.com/pubgo/dix/issues)
- 参与 [GitHub Discussions](https://github.com/pubgo/dix/discussions)

---

再次感谢您对 Dix 项目的贡献！🎉 