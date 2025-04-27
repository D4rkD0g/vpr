# VPR项目规划更新（2025-04-25）

注意严格按照目前 @types_v1.go 中的定义以及 @specification_v1.0.md 中的规范进行。注意在删除代码之前明确其是否有具体的含义以及是否会对当前逻辑有影响。

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
   - 实现核心Actions: http_request, wait, generate_data, authenticate
   - 实现特殊Actions: execute_local_commands, manual_action
   - 实现核心Checks: http_response_status, http_response_body, variable_equals等

3. **Phase 3（部分高级特性）** 
   - 实现部分提取器（Extractors）: extract_from_json, extract_from_body_regex, extract_from_header 
   - 实现重试和条件执行机制
   - 实现认证框架基础功能

### 最近完成的优化工作

1. **HTTP客户端统一管理**
   - 创建了统一的HTTP客户端工具（`pkg/utils/http.go`）
   - 修改了`ExecutionContext`，增加HTTP客户端的管理能力
   - 消除了多处重复定义`getHTTPClient`函数的问题
   - 确保在所有组件中使用一致的HTTP客户端接口
   - 通过重用客户端提高性能并保持cookie会话一致性

2. **重试机制优化**
   - 重构了重试逻辑框架，使其符合DSL v1.0规范
   - 区分了Actions的重试（retries, retry_delay）和Checks的轮询（max_attempts, retry_interval）
   - 创建了RetryableExecutor结构体，封装重试功能
   - 支持不同类型的重试策略（固定、指数、线性）
   - 提供了更优雅的错误处理和状态报告

3. **特殊动作类型实现**
   - `execute_local_commands` - 执行本地系统命令
   - `manual_action` - 用户交互和手动确认功能

## 完善计划（优化版）

根据DSL规范要求和项目审查，以下是更新后的完善计划，按实际优先级排序：

### 第一阶段：核心功能补全（预计1周）

#### 1. 提取器实现
- `extract_from_html` - HTML提取器
  - 功能描述: 使用CSS选择器从HTML内容提取数据
  - 优先级: 高（规范要求的核心功能）
  - 文件: `pkg/extractors/html.go`
  - 状态: [X] 已完成

- `extract_from_xml` - XML提取器
  - 功能描述: 使用XPath从XML内容提取数据
  - 优先级: 高（规范要求的核心功能）
  - 文件: `pkg/extractors/xml.go`
  - 状态: [X] 已完成

#### 2. HTTP功能增强
- `multipart` 表单处理
  - 功能描述: 支持multipart/form-data格式，用于文件上传等
  - 优先级: 高（许多漏洞利用需要文件上传功能）
  - 文件: `pkg/actions/http_multipart.go`
  - 状态: [X] 已完成

#### 3. 变量函数实现
- 内置变量函数系统
  - 功能描述: 实现规范要求的变量函数（base64, url, json处理等）
  - 优先级: 高（规范要求的核心功能）
  - 文件: `pkg/context/functions.go`
  - 状态: [X] 已完成

### 第二阶段：测试和验证（当前进行中）

#### 1. 测试核心功能
- 编写测试代码验证完整PoC功能
  - 功能描述: 加载example目录中的PoC文件，对指定目标进行验证测试
  - 优先级: 高（验证已实现功能的正确性）
  - 计划:
    - [ ] 创建测试PoC文件（如果examples目录中没有适合的）
    - [ ] 修改main.go加载和运行PoC
    - [ ] 运行测试并验证结果

#### 2. 远程资源检查
- `check_remote_resource` - 远程资源检查
  - 功能描述: 检查远程服务器上的资源（文件、目录等）
  - 优先级: 中
  - 文件: `pkg/checks/remote.go`
  - 计划:
    - [ ] 实现远程资源检查逻辑
    - [ ] 编写测试代码验证功能正确性

#### 3. 凭证管理增强
- 环境变量凭证解析器
  - 功能描述: 从环境变量获取凭证信息
  - 优先级: 中
  - 文件: `pkg/credentials/environment.go`
  - 计划:
    - [ ] 实现环境变量凭证解析逻辑
    - [ ] 编写测试代码验证功能正确性

- 文件凭证解析器
  - 功能描述: 从配置文件获取凭证信息
  - 优先级: 中
  - 文件: `pkg/credentials/file.go`
  - 计划:
    - [ ] 实现文件凭证解析逻辑
    - [ ] 编写测试代码验证功能正确性

### 第三阶段：CLI与发布（预计1周）

#### 1. CLI 界面完善
- 命令行参数处理
  - 功能描述: 支持配置文件、命令行参数等
  - 优先级: 中
  - 文件: `cmd/vpr/main.go`
  - 计划:
    - [ ] 实现命令行参数解析逻辑
    - [ ] 编写测试代码验证功能正确性

- 交互式模式
  - 功能描述: 支持交互式执行和调试
  - 优先级: 中低
  - 文件: `cmd/vpr/interactive.go`
  - 计划:
    - [ ] 实现交互式模式逻辑
    - [ ] 编写测试代码验证功能正确性

#### 2. 文档和发布
- 使用文档
  - 功能描述: 编写详细的使用文档和API参考
  - 优先级: 中
  - 文件: `docs/usage.md`, `docs/api_reference.md`
  - 计划:
    - [ ] 编写使用文档
    - [ ] 编写API参考文档

- 发布准备
  - 功能描述: 版本标记、CHANGELOG、安装说明等
  - 优先级: 中
  - 文件: `CHANGELOG.md`, `README.md`, `INSTALL.md`
  - 计划:
    - [ ] 更新版本标记
    - [ ] 编写CHANGELOG
    - [ ] 编写安装说明

## 已完成的修复

1. **解决处理程序重复注册问题**：
   - [X] 修复`http_request`处理程序在多个文件中重复注册的问题
   - [X] 修复`authenticate`处理程序重复注册问题
   - [X] 修复`generate_data`处理程序重复注册问题
   - [X] 修复`wait`处理程序重复注册问题

2. **修复YAML测试文件**：
   - [X] 优化`examples/test_features.yaml`文件格式，符合DSL规范
   - [X] 修复变量引用格式，使用正确的路径和字段名大小写
   - [X] 将复杂的command结构简化为简单的字符串数组

## 当前状态

我们已经修复了许多重复注册问题和YAML文件格式问题，现在需要启动测试服务器后再运行测试PoC。测试服务器定义在`examples/test_server.go`，用于提供HTML、XML和文件上传测试接口。

## 后续步骤

1. **测试服务器启动**：
   - [ ] 启动`test_server.go`作为后台服务
   - [ ] 确保服务器在localhost:8080上监听

2. **运行测试PoC**：
   - [ ] 使用修复后的`test_features.yaml`运行测试
   - [ ] 解决任何剩余的执行问题

3. **其他规划中的功能实现**：
   - [ ] 实现远程资源检查
   - [ ] 实现凭证管理
   - [ ] 完善CLI接口

## 开发原则（继续遵循）

1. **代码变更原则**：
   - 在删除任何代码前，先理解其功能和目的
   - 确保新实现能完全替代原有功能后再删除
   - 对于重构部分，保留详细注释说明变更原因

2. **规范遵循原则**：
   - 严格按照`specification_v1.0.md`规范进行实现
   - 使用`types_v1.go`中定义的数据结构确保类型安全
   - 保持与现有代码一致的风格和命名约定

3. **测试需求**：
   - 为每个新增或修改的组件编写单元测试
   - 确保新功能不会破坏现有功能
   - 创建端到端测试场景验证完整流程

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

## 关键设计决策与原则

1. **模块化与可扩展性**
   - 使用组件注册模式，便于添加新的Actions/Checks/Extractors
   - 清晰的关注点分离，每个包负责特定功能
   - 插件化架构，支持未来扩展

2. **变量系统设计**
   - 统一变量解析，支持嵌套引用和函数调用
   - 上下文隔离，确保变量作用域明确
   - 完整的类型转换和验证

3. **错误处理策略**
   - 区分预期错误和非预期错误
   - 结构化错误上下文，便于分析和报告
   - 灵活的重试策略，处理不同类型的失败

4. **安全考量**
   - 凭证管理安全性，避免硬编码敏感信息
   - 资源利用限制，防止DoS风险
   - 命令执行沙箱化，限制潜在影响

5. **测试与质量保证**
   - 单元测试覆盖核心功能
   - 集成测试验证端到端流程
   - 示例PoC作为功能测试用例

## 开发教训与最佳实践

1. **代码重用与整合**
   - 在开发新功能前检查现有实现，避免代码重复
   - 使用包装器模式保留现有功能同时增强能力
   - 统一配置和客户端管理，确保一致性

2. **错误处理模式**
   - 结构化错误信息，包含足够上下文
   - 区分不同错误类型并采用相应处理策略
   - 使用统一的日志模式，便于故障排查

3. **规范遵循**
   - 严格按照`specification_v1.0.md`规范进行实现
   - 使用`types_v1.go`中定义的数据结构确保类型安全
   - 保持与现有代码一致的风格和命名约定

## 里程碑计划（更新版）

### 里程碑1：核心功能完成
- 预计完成时间: 2025-05-08
- 完成提取器功能（HTML/XML）
- 实现multipart表单支持
- 完成变量函数系统

### 里程碑2：高级功能与测试
- 预计完成时间: 2025-05-15
- 完成远程资源检查
- 增强凭证管理
- 实现完整测试覆盖

### 里程碑3：CLI与发布
- 预计完成时间: 2025-05-25
- 完成CLI界面
- 完善文档
- 发布v1.0稳定版

## 特殊动作类型实现更新 (2025-04-25)

### 今日完成的工作

今天我们完成了两个特殊动作类型的实现，这些功能对于提高VPR的交互性和灵活性非常重要：

### 1. 本地命令执行 (`execute_local_commands`)
- 已实现 `pkg/actions/local_commands.go`
- 功能：允许PoC执行本地系统命令
- 主要特性：
  - 支持依次执行多个命令
  - 捕获命令输出和退出代码
  - 变量替换支持，使命令可动态生成
  - 结果存储到目标变量
  - 详细的结构化日志记录
  - 安全考量：执行结果包含退出码、输出和错误信息

### 2. 手动操作动作 (`manual_action`)
- 已实现 `pkg/actions/manual.go`
- 功能：暂停执行流程，等待用户交互和确认
- 主要特性：
  - 自定义提示信息支持
  - 超时机制，避免无限等待
  - 用户友好的控制台交互界面
  - 支持yes/no确认
  - 结果存储到目标变量
  - 完整的时间跟踪

## VPR修复进度 - 2025-04-27

### 当前任务
修复VPR PoC执行问题，特别是文件上传和变量替换问题

### 已解决问题
- [X] 修复了重复注册的动作处理程序
- [X] 修复了YAML解析逻辑
- [X] 修复了YAML字段名称不匹配问题
- [X] 修复了`generate_data`动作缺少参数的问题

### 当前问题
- [ ] 文件上传失败 - 找不到测试文件
  - 问题：尽管setup阶段显示文件创建成功，但exploit阶段无法找到文件
  - 尝试方案：使用绝对路径创建和引用文件
- [ ] 完成剩余的测试步骤

### 下一步
1. 修改test_features.yaml使用绝对路径
2. 确保文件创建和访问正常
3. 验证整个PoC执行流程
4. 检查剩余的任何问题

### Lessons
- 使用`generate_data`动作时需要提供`parameters`字段和`type`参数
- 文件路径在VPR中应尽量使用绝对路径，因为相对路径可能受工作目录影响
- 模板变量（如`{{ ... }}`）在shell命令中可能不会被正确替换

## 当前PoC测试问题和修复（2025-04-27）

### 所有问题已解决：

[X] 修复了LoadPocFromFile函数中的YAML解析逻辑，正确处理DSL版本验证
[X] 修复了test_features.yaml中的命令结构，将execute_local_commands的命令修改为字符串数组
[X] 更新了variable_equals检查，使用path字段而不是variable字段指向变量路径
[X] 修改test_features.yaml中的变量引用格式，使用正确的字段名称大小写
[X] 修复generate_data动作定义，添加了必需的parameters字段和type参数
[X] 变量设置问题：修改了generateDataHandler以创建匹配ContextVariable格式的变量结构，并使用正确的路径格式存储变量
[X] 修改了HTML和XML extractors以正确地创建ContextVariable结构并使用正确的路径格式存储变量
[X] 修改了httpRequestHandler函数，添加对response_actions的处理逻辑
[X] 实现了提取器注册表（ExtractorRegistry）及其初始化函数
[X] 修复了JSON提取器中的变量存储逻辑
[X] 修复了正则表达式提取器中的变量存储逻辑
[X] 实现了Header提取器
[X] 修复了导入循环问题
[X] 修复了XML提取器中的属性处理问题，正确提取属性值
[X] 修改httpRequestHandler函数，将HTTP响应存储在上下文中供HTTP响应检查使用
[X] 文件上传问题：成功解决，现在系统可以正确处理文件上传测试

### 学习要点：

- 变量引用中区分大小写，如.Value而不是.value
- 变量存储为完整的结构体，包含ID和Value字段
- 变量访问路径应使用完整格式：variables.{variable_name}.Value
- 检查时使用path字段指向变量路径而不是variable字段
- 确保HTTP response actions在httpRequestHandler中被正确处理
- 提取器需要正确地创建变量并将其存储在上下文中
- 在处理XPath表达式时，当选择属性节点时需特别注意提取属性值而非属性名
- HTTP响应检查需要访问上下文中存储的HTTP响应数据，每次HTTP请求后需要更新这个数据
- 避免包之间的循环依赖，保持合理的代码结构
- 确保重要的状态（如HTTP响应）在执行结束前被正确保存

### VPR执行流程总结

1. 加载和解析PoC定义文件（YAML）
2. 创建执行上下文，初始化各种处理程序和注册表
3. 执行Setup阶段，建立执行环境
4. 执行Exploit阶段，执行核心操作序列
5. 执行Assertions阶段，验证操作结果
6. 执行Verification阶段，进行最终验证和清理

### 关键组件职责

- **context包**：管理执行状态，提供变量存储和访问功能
- **actions包**：实现各种动作处理程序（HTTP请求、本地命令等）
- **extractors包**：从响应中提取数据（HTML、XML、JSON等）
- **checks包**：实现各种断言检查（变量比较、HTTP响应检查等）
- **executor包**：编排整个执行流程，管理阶段和步骤执行

### 所有测试现在都通过了！🎉

## 新增功能（2025-04-27）

今天新增了一个实用的功能：

### 命令行目标覆盖

在`cmd/vpr/main.go`中添加了命令行参数，允许用户在运行时覆盖PoC定义中的目标设置：

- `-host`：覆盖目标主机（例如：`-host example.com`）
- `-port`：覆盖目标端口（例如：`-port 8443`）
- `-url`：覆盖完整的目标URL（例如：`-url https://example.com:8443`）

这些参数会优先于PoC文件中定义的值，使测试更加灵活。用法示例：

```bash
go run cmd/vpr/main.go -p examples/test_features.yaml -host localhost -port 9090
```

### 下一步功能增强计划

1. 添加更多命令行选项：
   - 凭证管理选项
   - 超时设置
   - 详细输出控制

2. 实现结果输出格式化：
   - JSON输出
   - HTML报告生成
   - 结果摘要

3. 完善错误处理和用户反馈
