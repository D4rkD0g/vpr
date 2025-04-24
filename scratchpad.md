# VPR项目规划更新（2025-04-24）

## 项目概述

VPR（Vulnerability Proof-of-Concept Runner）是一个基于Go的执行引擎，用于执行符合DSL v1.0规范的安全PoC（Proof of Concept）定义。该项目旨在提供一个标准化的框架，使安全研究人员能够以可重复、可验证的方式描述和执行漏洞利用过程。

## 项目进展

### 已完成阶段

1. **Phase 1（核心基础架构）** 
   - 实现YAML解析器和验证器 
   - 构建上下文管理系统和变量解析
   - 创建组件注册系统（actions, checks, extractors）
   - 设计执行引擎架构

2. **Phase 2（基本执行功能）** 
   - 实现执行引擎及阶段顺序（setup → exploit → assertions → verification）
   - 创建步骤执行器，支持条件执行（if）和循环（loop）
   - 构建HTTP客户端包装器及安全控制
   - 实现核心Actions: http_request, wait, generate_data
   - 实现核心Checks: http_response_status, http_response_body, variable_equals等

3. **Phase 3（高级特性）** 
   - 实现提取器（Extractors）: extract_from_json, extract_from_body_regex, extract_from_header 
   - 添加内置函数 
   - 实现认证框架基础功能 

## 完善计划（优化版）

根据DSL规范要求和项目审查，以下是优化后的完善计划：

### 第一阶段：核心功能补全（预计1-2周）

#### 1. 关键动作类型实现
- [ ] `check_target_availability` - 目标可用性检查
  - 功能描述: 验证目标系统是否可访问，支持HTTP/HTTPS/TCP检查
  - 优先级: 高（作为setup阶段的前置检查）
  - 文件: `pkg/actions/availability.go`

- [ ] `authenticate` - 专用认证动作
  - 功能描述: 实现完整的认证流程，支持OAuth2, Form等多种认证方式
  - 优先级: 高（许多PoC依赖此功能）
  - 文件: `pkg/actions/authentication.go`

- [ ] `ensure_resource_exists` - 资源存在确认
  - 功能描述: 确保特定资源存在，如必要时创建
  - 优先级: 中高
  - 文件: `pkg/actions/resources.go`

#### 2. 高级错误处理机制
- [ ] `expected_error` 处理功能
  - 功能描述: 支持指定预期的错误条件，区分预期失败和非预期失败
  - 优先级: 高（对测试负面场景至关重要）
  - 文件: `pkg/checks/errors.go`

- [ ] 完善重试策略
  - 功能描述: 增强重试机制，支持更灵活的重试策略（指数退避、条件重试等）
  - 优先级: 中高
  - 文件: `pkg/executor/retry.go`

#### 3. 高级检查类型
- [ ] `json_schema_validation` - JSON Schema验证
  - 功能描述: 使用JSON Schema验证响应内容
  - 优先级: 中高
  - 文件: `pkg/checks/json_schema.go`

- [ ] `check_remote_resource` - 远程资源检查
  - 功能描述: 检查远程服务器上的资源（文件、目录等）
  - 优先级: 中
  - 文件: `pkg/checks/remote.go`

### 第二阶段：功能增强与完善（预计1-2周）

#### 1. 凭证管理增强
- [ ] 环境变量凭证解析器
  - 功能描述: 从环境变量获取凭证信息
  - 优先级: 中
  - 文件: `pkg/credentials/environment.go`

- [ ] 文件凭证解析器
  - 功能描述: 从配置文件获取凭证信息
  - 优先级: 中
  - 文件: `pkg/credentials/file.go`

#### 2. HTTP功能增强
- [ ] `multipart` 表单处理
  - 功能描述: 支持multipart/form-data格式，用于文件上传等
  - 优先级: 中
  - 文件: `pkg/actions/http_multipart.go`

#### 3. 补充提取器
- [ ] `extract_from_html` - HTML提取器
  - 功能描述: 使用CSS选择器从HTML内容提取数据
  - 优先级: 中
  - 文件: `pkg/extractors/html.go`

- [ ] `extract_from_xml` - XML提取器
  - 功能描述: 使用XPath从XML内容提取数据
  - 优先级: 中低
  - 文件: `pkg/extractors/xml.go`

#### 4. 特殊动作类型
- [ ] `execute_local_commands` - 本地命令执行
  - 功能描述: 执行本地系统命令
  - 优先级: 中低（安全考量）
  - 文件: `pkg/actions/local_commands.go`

- [ ] `manual_action` - 手动操作动作
  - 功能描述: 支持用户交互，等待手动确认
  - 优先级: 中
  - 文件: `pkg/actions/manual.go`

### 第三阶段：集成与测试（预计1周）

#### 1. CLI 界面完善
- [ ] 命令行参数处理
  - 功能描述: 支持配置文件、命令行参数等
  - 优先级: 中
  - 文件: `cmd/vpr/main.go`

- [ ] 交互式模式
  - 功能描述: 支持交互式执行和调试
  - 优先级: 中低
  - 文件: `cmd/vpr/interactive.go`

#### 2. 测试与文档
- [ ] 单元测试覆盖
  - 功能描述: 为主要组件编写全面的单元测试
  - 优先级: 高
  - 文件: 各组件对应的`*_test.go`文件

- [ ] 示例PoC编写
  - 功能描述: 创建涵盖各种场景的示例PoC
  - 优先级: 中高
  - 文件: `examples/`目录

- [ ] 使用文档
  - 功能描述: 编写详细的使用文档和API参考
  - 优先级: 中
  - 文件: `docs/usage.md`, `docs/api_reference.md`

## 项目架构与模块关系

```
vpr/
├── cmd/                # 命令行工具
│   └── vpr/            # 主程序
├── pkg/                # 核心库
│   ├── actions/        # 动作处理器
│   ├── checks/         # 检查处理器
│   ├── context/        # 上下文管理
│   ├── credentials/    # 凭证管理
│   ├── executor/       # 执行引擎
│   ├── extractors/     # 数据提取器
│   ├── poc/            # PoC定义和解析
│   ├── reporting/      # 结果报告
│   └── utils/          # 通用工具
├── examples/           # 示例PoC
├── docs/               # 文档
│   └── dsl/            # DSL规范
└── tests/              # 测试文件
```

## 技术决策与设计原则

1. **模块化设计**
   - 使用组件注册模式，便于扩展新的Actions/Checks/Extractors
   - 清晰的关注点分离，每个包负责特定功能

2. **变量解析系统**
   - 灵活的模板替换，支持嵌套变量和函数调用
   - 内置函数体系，支持常见操作和转换

3. **安全设计**
   - 凭证引用而非直接存储敏感信息
   - 可配置的安全限制（如禁止任意命令执行）

4. **执行流程设计**
   - 严格遵循DSL规范定义的阶段顺序
   - 支持条件执行和循环，增强灵活性

5. **错误处理与日志**
   - 详细的执行结果报告，区分不同类型的失败
   - 结构化日志，便于故障排查和分析

## 项目风险与挑战

1. **安全考量**
   - 执行未知PoC的潜在风险
   - 凭证处理的安全性

2. **兼容性**
   - 确保与各种HTTP行为和响应格式兼容
   - 处理不同环境下的差异

3. **可维护性**
   - 平衡功能丰富性和代码复杂度
   - 保持良好的测试覆盖率

## 团队分工与里程碑

### 里程碑1（核心功能补全）
- 预计完成时间: 2025-05-08
- 优先实现高优先级动作和检查类型
- 完成错误处理机制基础框架

### 里程碑2（功能增强）
- 预计完成时间: 2025-05-22
- 完成所有剩余提取器和动作类型
- 增强凭证管理系统

### 里程碑3（稳定版发布）
- 预计完成时间: 2025-05-29
- 完成CLI界面和文档
- 发布v1.0稳定版
