# AI代码检查工具

这是一个基于AI的代码质量检查工具，支持多种AI服务提供商，可以根据自定义规则对指定目录中的代码文件进行自动化检查和分析。

## 功能特性

- 🤖 **多AI服务支持**：支持OpenAI、SiliconFlow、AiHubMix、火山引擎等多种AI服务
- 📝 **自定义规则**：通过JSON配置文件定义检查规则，支持文件类型过滤和关键字匹配
- 🚀 **并发处理**：支持多任务并发执行，大幅提升检查效率
- 📊 **详细报告**：生成Markdown格式的检查报告，支持SVN日志集成
- ⏸️ **断点续检**：自动跳过已检查的文件，支持中断后继续检查
- 🔄 **分片处理**：自动将大文件分片处理，避免API限制
- 📋 **SVN集成**：自动获取文件SVN提交历史和主要作者信息
- ⏰ **时间过滤**：可配置只检查指定时间之后有SVN提交的文件，聚焦最近修改

## 使用

### 1. 配置文件

创建 `config.json` 配置文件（必需）：

```json
{
    "api": {
        "type": "volcengine",
        "url": "https://ark.cn-beijing.volces.com/api/v3/chat/completions",
        "key": "your-api-key",
        "model": "your-model-name",
        "max_tokens": 32000,
        "enable_log": true,
        "max_text_length": 64000
    },
    "check": {
        "directory": "/path/to/check",
        "output_dir": "./check_results",
        "concurrency": 10
    },
    "svn": {
        "log_limit": 50,
        "priority_authors": ["author1", "author2"],
        "filter_after": "2025-05-01T00:00:00Z"
    },
    "rules": [
        {
            "name": "通用代码检查",
            "description": "检查代码中的潜在问题，包括逻辑错误、安全隐患、性能问题等",
            "extensions": [".lua", ".js", ".py", ".cpp", ".java"],
            "keywords": [],
            "enabled": true
        },
        {
            "name": "特定API使用检查",
            "description": "检查特定API的使用是否规范，是否存在错误用法",
            "extensions": [".lua", ".js"],
            "keywords": ["特定API名称"],
            "enabled": false
        }
    ]
}
```

### 2. 运行检查

```bash
# 使用默认配置文件 config.json
./code-checker.exe

# 使用指定配置文件
./code-checker.exe -config config.json
```

## 详细配置说明

### API配置 (`api`)

| 参数 | 类型 | 说明 |
|------|------|------|
| `type` | string | AI服务提供商类型: "openai","siliconflow","aihubmix","volcengine" |
| `url` | string | API服务地址 |
| `key` | string | API密钥 |
| `model` | string | 使用的AI模型 |
| `max_tokens` | int | API返回的最大token数 |
| `enable_log` | bool | 是否启用API请求日志 |
| `max_text_length` | int | 单次请求最大文本长度（字符数） |

#### 支持的AI服务类型

| type值 | 服务商 | 说明 |
|--------|--------|------|
| `openai` | OpenAI | 官方OpenAI或兼容接口 |
| `siliconflow` | SiliconFlow | SiliconFlow平台 |
| `aihubmix` | AiHubMix | AiHubMix平台 |
| `volcengine` | 火山引擎 | 字节跳动火山引擎 |

### 检查配置 (`check`)

| 参数 | 类型 | 说明 |
|------|------|------|
| `directory` | string | 要检查的目录路径 |
| `output_dir` | string | 检查结果输出目录 |
| `concurrency` | int | 并发检查任务数量 |

### SVN配置 (`svn`)

| 参数 | 类型 | 说明 |
|------|------|------|
| `log_limit` | int | SVN日志获取的最大记录数 |
| `priority_authors` | []string | 优先级作者列表，用于确定文件主要负责人 |
| `filter_after` | string | 时间过滤，格式：2024-01-01T00:00:00Z，只检查该时间之后有提交的文件，留空则不启用 |

### 规则配置 (`rules`)

规则配置是一个数组，每个规则包含以下字段：

| 参数 | 类型 | 说明 |
|------|------|------|
| `name` | string | 规则名称，用作输出文件夹名 |
| `description` | string | 规则描述，作为AI检查的提示词 |
| `extensions` | []string | 要检查的文件扩展名列表（如 `[".lua", ".js"]`） |
| `keywords` | []string | 关键字过滤，只检查包含这些关键字的文件 |
| `enabled` | bool | 是否启用此规则 |

### 规则匹配逻辑

1. **文件扩展名匹配**：文件扩展名必须在规则的 `extensions` 列表中
2. **关键字匹配**：如果规则设置了 `keywords`，文件内容必须包含至少一个关键字
3. **规则启用状态**：只有 `enabled` 为 `true` 的规则才会执行

## 输出结果

检查完成后，会在 `output_dir` 目录下生成以下结构：

```
check_results/
├── 规则名称1/
│   ├── [作者]文件名1.扩展名.md
│   ├── [作者]文件名2.扩展名.md
│   └── ...
├── 规则名称2/
│   ├── [作者]文件名1.扩展名.md
│   └── ...
└── ...
```

每个Markdown文件包含：
- 文件路径和基本信息
- SVN提交历史（最近N条记录）
- 主要作者信息
- AI检查结果和建议

## 高级功能

### 1. 断点续检

工具会自动检查输出目录中已存在的结果文件，跳过已检查的内容。如果需要重新检查，请删除对应的结果文件。

### 2. 大文件分片

当文件内容超过 `max_text_length` 设置时，工具会自动将文件按行分片处理，然后合并检查结果。

### 3. 并发优化

通过调整 `concurrency` 参数可以控制并发任务数量：
- 值过小：检查速度慢
- 值过大：可能触发API限制或占用过多系统资源
- 建议根据API限制和系统性能设置为 3-20

### 4. 日志功能

设置 `enable_log: true` 会在 `logs/` 目录下记录所有API请求和响应，便于调试和问题排查。

### 5. SVN时间过滤

通过设置 `svn.filter_after` 参数，可以只检查在指定时间之后有SVN提交的文件：

- **时间格式**：使用RFC3339格式，如 `2024-12-01T00:00:00Z`
- **过滤逻辑**：工具会检查每个文件的SVN提交历史，只有在指定时间之后有提交记录的文件才会被检查
- **用途**：适合增量检查，只关注最近修改的代码文件
- **错误处理**：如果SVN检查失败（如文件不在版本控制下），默认允许检查该文件

**示例配置**：
```json
{
    "svn": {
        "filter_after": "2024-12-01T00:00:00Z"  // 只检查12月1日之后修改的文件
    }
}
```

## 常见问题

### Q: 如何自定义检查规则？

**A**: 修改 `config.json` 文件中的 `rules` 数组：
1. 在 `description` 中详细描述要检查的内容
2. 设置合适的 `extensions` 文件类型
3. 使用 `keywords` 精确筛选目标文件
4. 可以设置 `enabled: false` 临时禁用某个规则

### Q: 如何只检查最近修改的文件？

**A**: 使用SVN时间过滤功能：
1. 在 `config.json` 中设置 `svn.filter_after` 参数
2. 使用RFC3339时间格式，如 `2024-12-01T00:00:00Z`
3. 工具会自动跳过在指定时间之前最后一次提交的文件
4. 适合定期检查或CI/CD流水线中的增量检查

