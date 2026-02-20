---
name: stp
description: 结构化任务规划与分步执行（Structured Task Planning）。支持两种模式：(1) 文件模式 - 从 Markdown 任务文档加载步骤执行；(2) 自然语言模式 - 接受自然语言任务描述，自动生成计划书并确认后执行。触发词：/stp、任务规划、步骤执行。功能包括：步骤分解、状态跟踪、执行日志记录、快速失败策略。
description_en: Structured Task Planning with step-by-step execution. Supports two modes: (1) File Mode - load steps from Markdown task documents; (2) Natural Language Mode - accept natural language descriptions, auto-generate plan, and execute after user confirmation. Triggers: /stp, "任务规划", "步骤执行". Features: step decomposition, status tracking, execution logging, fast-fail strategy.
---

> **路径变量说明**（本文档通用）：
> - `<STP_ROOT>` = `~/.openclaw/workspace/skills/stp`
> - `<STP_SCRIPTS>` = `<STP_ROOT>/scripts`
> - `<STP_TASK_LIST>` = `~/.openclaw/workspace/task-list`
> - `<STP_TASKS>` = `~/.openclaw/workspace/tasks`

# STP（Structured Task Planning）结构化任务规划与分步执行

## 功能概述

1. **任务解析**：从 Markdown 任务文档加载步骤
2. **确认流程**：生成计划书后必须用户确认才能执行（所有模式）
3. **目录管理**：自动创建任务专属目录（格式：`task-{ID}`）
4. **文档生成**：生成标准化步骤文档和执行日志
5. **状态跟踪**：支持步骤状态标记（待执行/成功/失败）
6. **智能执行**：每个步骤作为子任务执行，附带成功判断标准
7. **快速失败**：步骤失败立即终止，不尝试替代方案
8. **完整日志**：所有 exec 命令和 AI 执行过程完整记录

---

## 两种使用模式

### 模式 A：文件模式（用户提供 Markdown 文档）

用户提供已写好的任务文档路径：
```bash
# 完整参数
python3 <STP_SCRIPTS>/execute_task.py --file /path/to/task.md

# 简写参数
python3 <STP_SCRIPTS>/execute_task.py -f /path/to/task.md
```

**流程**：
1. 读取 Markdown 任务文档
2. 解析任务名称和执行步骤
3. 生成计划书并展示
4. **等待用户确认** ← 必须确认才能继续
5. 用户确认后创建任务目录并执行

---

### 模式 B：自然语言模式（AI 自动生成计划）

用户用自然语言描述任务，AI 自动生成计划书：
```bash
# 完整参数
python3 <STP_SCRIPTS>/execute_task.py --nlp "我想查一下腾讯、贵州茅台、Meta 的股票价格"

# 简写参数
python3 <STP_SCRIPTS>/execute_task.py -n "帮我安装 CosyVoice"
```

**流程**：
1. 接收自然语言任务描述
2. AI 分析意图并生成 Markdown 计划书
3. 保存计划书到 `<STP_TASK_LIST>/<filename>.md`
4. 生成计划书并展示
5. **等待用户确认** ← 必须确认才能继续
6. 用户确认后开始执行

---

## 两种模式的核心区别

| 特性 | 模式 A：文件模式 | 模式 B：自然语言模式 |
|------|------------------|----------------------|
| **输入** | 已有 `.md` 文件 | 自然语言描述 |
| **计划书来源** | 用户提供 | AI 自动生成 |
| **适用场景** | 复杂/重复任务（复用计划） | 临时/简单任务 |
| **参数** | `--file` / `-f` | `--nlp` / `-n` |
| **确认流程** | ✅ 都有 | ✅ 都有 |

**重要提示**：
- 两种模式**都必须**经过用户确认才能执行
- 参数不区分大小写，但建议统一使用小写
- 混用参数时，`--file` 优先于 `--nlp`

---

## 模式 B：自然语言任务规划流程（Agent 专用）

当用户输入自然语言描述时，主 Agent 需要：

### 第一步：任务分析与计划生成

1. **理解用户意图**：分析自然语言，明确任务目标
2. **拆解执行步骤**：将任务分解为可执行的具体步骤
3. **生成计划书**：按标准格式生成 Markdown 任务文档
4. **保存到 task-list**：`~/.openclaw/workspace/task-list/<filename>.md`

### 第二步：展示并确认

```markdown
📋 任务计划书已生成

任务名称：xxx
文件位置：~/.openclaw/workspace/task-list/xxx.md
步骤数：5

核心执行步骤：
- [ ] 步骤 1：xxx
- [ ] 步骤 2：xxx
...

确认执行：
  输入 "ok" 或 "确认" → 开始执行
  输入 "取消" → 放弃此任务
  输入修改意见 → 我会调整计划后重新展示
```

### 第三步：执行或取消

- **确认后**：保存任务文档，调用 STP 脚本开始执行
- **取消**：不执行，记录日志
- **修改**：更新计划书后重新展示

---

## 计划书标准格式

```markdown
# 任务名称

## 任务描述
用户原始需求的简要描述

## 技术方案
- 使用的工具/库/API
- 关键技术约束

## 核心执行步骤
- [ ] 步骤 1：具体描述（含成功标准）
- [ ] 步骤 2：具体描述（含成功标准）
- [ ] 步骤 3：具体描述（含成功标准）

## 预期产出
- 输出文件/结果说明
```

### 示例计划书

```markdown
# 三支股票收盘价查询

## 任务描述
查询腾讯控股、贵州茅台、Meta 三支股票的最新收盘价

## 技术方案
- 使用 AkShare 库查询股票数据
- 支持 A股(沪深)、港股、美股三个市场

## 核心执行步骤
- [ ] 步骤 1：编写基于 AkShare 的股票查询脚本，保存至 `temp/scripts/stock_query.py`
- [ ] 步骤 2：查询贵州茅台（600519.SH）收盘价
- [ ] 步骤 3：查询腾讯控股（00700.HK）收盘价
- [ ] 步骤 4：查询 Meta（META.O）收盘价

## 预期产出
- 三支股票的收盘价信息（股票名称、代码、日期、价格、市场）
```

---

## 模式 A：文件模式使用方式

### 步骤1：准备 Markdown 任务文档

```markdown
# 任务名称

## 核心执行步骤
- [ ] 步骤 1：具体描述
- [ ] 步骤 2：具体描述
- [ ] 步骤 3：具体描述
```

### 步骤2：运行并确认

```bash
python3 <STP_SCRIPTS>/execute_task.py --file /path/to/task.md
```

**输出示例**：
```
========================================
📋 任务计划书已生成
========================================

任务名称：安装 Conda 环境
文件：/Users/scotthuang/task.md
步骤数：4

核心执行步骤：
  - [ ] 步骤 1：下载 Miniforge3
  - [ ] 步骤 2：运行安装脚本
  - [ ] 步骤 3：验证 conda 安装
  - [ ] 步骤 4：创建 cosyvoice 环境

========================================
确认执行：
  输入 "ok" 或 "确认" → 开始执行
  输入 "取消" → 放弃此任务
  输入修改意见 → 我会调整后重新展示
========================================
```

### 步骤3：用户确认后自动执行

- ✅ 确认 → 创建任务目录，开始执行
- ❌ 取消 → 终止，不创建任务
- 📝 修改 → 更新文档后重新展示

---

## Agent 执行规范（重要！）

执行每个步骤时，**必须**记录关键命令和输出：

### 方法：步骤完成后补记执行日志

1. **执行步骤时**：正常用 `exec` tool 执行命令
2. **步骤完成后**：把关键命令和输出写入临时文件，然后用 `--exec-file` 记录

```bash
# 1. 先把执行记录写入临时文件
cat > /tmp/step_exec.log << 'EOF'
[命令1] python3 -c "import akshare; print(akshare.__version__)"
[输出1] AkShare version: 1.18.22
[退出码] 0

[命令2] python3 stock_query.py 600519.SH
[输出2] 贵州茅台 收盘价: 1515.01
[退出码] 0
EOF

# 2. 记录步骤状态时附带执行日志
python3 execute_task.py --log task-8 1 success "脚本执行成功" --exec-file /tmp/step_exec.log
```

### 执行日志格式建议

```
[命令] <执行的完整命令>
[输出] <命令输出（可截断关键部分）>
[退出码] <0=成功, 非0=失败>
```

---

## 任务目录结构

```
~/.openclaw/workspace/tasks/
└── task-XXX/
    ├── task_steps.md      # 步骤文档（含状态标记和成功标准）
    ├── task_execution.log # 完整执行日志（含所有exec命令）
    ├── result.txt         # 结果汇总
    └── temp/
        ├── scripts/
        └── downloads/
```

### task-list 目录

```
~/.openclaw/workspace/task-list/
├── stock-query-20260208.md    # 自然语言模式生成的计划书
├── highway-query-20260208.md   # ...
└── ...
```

---

## 状态标记

| 标记 | 含义 |
|------|------|
| `[ ]` | 待执行 |
| `✓` | 执行成功 |
| `✗` | 执行失败（任务终止） |

---

## 执行规则（重要！）

### 1. 原方案优先原则
**必须严格按照任务文档中定义的方案执行，禁止擅自更改实现方式。**

示例：
```
❌ 错误做法：原方案要求用 AkShare，实际改用新浪 API
✅ 正确做法：严格使用 AkShare，遇到问题按失败处理
```

### 2. 快速失败策略

**原则**：一步到位，不做替代方案尝试

- 子任务返回「失败」状态 → 立即终止整个任务链
- 不自动重试、不尝试其他实现方式
- 记录失败原因到日志

### 3. 失败处理
- 方案执行失败 → 立即终止，不尝试替代方案
- 明确记录失败原因（违反方案/技术限制/接口问题）
- 任务结果标记为 FAILED

---

## 资源

### scripts/
- `execute_task.py` - 主脚本
  - `--file <path>` / `-f <path>`：文件模式
  - `--nlp <text>` / `-n <text>`：自然语言模式
  - `--subtask <task_id> <step_num>`：生成子任务 prompt
  - `--log <task_id> <step> <status> [消息]`：记录步骤状态
  - `--exec <task_id> "<命令>" <exit_code> [输出]`：记录 exec
  - `--exec-file <path>`：从文件读取执行日志（与 --log 配合）
  - 自动记录 exec 命令到日志
  - 自动创建 `.task_counter` 自增ID

### task-list/
- 存放自然语言模式生成的任务计划书
- 路径：`<STP_TASK_LIST>`
- 文件命名格式：`<简短描述>-<YYYYMMDD>.md`

---

## 使用示例

### 示例 1：文件模式（推荐用于复杂任务）

```bash
# 准备任务文件
cat > ~/task-docs/cosyvoice-install.md << 'EOF'
# Mac M系列芯片安装CosyVoice

## 技术方案
- 使用 Conda 管理环境
- CPU 模式运行（PyTorch）

## 核心执行步骤
- [ ] 步骤 1：下载 Miniforge3 安装脚本
- [ ] 步骤 2：运行安装脚本
- [ ] 步骤 3：验证 conda 安装
- [ ] 步骤 4：创建 cosyvoice 环境
EOF

# 执行（会先展示计划书并等待确认）
python3 <STP_SCRIPTS>/execute_task.py -f ~/task-docs/cosyvoice-install.md
```

### 示例 2：自然语言模式（推荐用于临时任务）

```bash
# 直接描述需求
python3 <STP_SCRIPTS>/execute_task.py -n "帮我安装 CosyVoice"
```

### 示例 3：Agent 集成（主 Agent 调用）

```python
# 用户输入自然语言
user_input = "帮我查一下腾讯和茅台的股票价格"

# Agent 调用自然语言模式
import subprocess
result = subprocess.run([
    'python3', '<STP_SCRIPTS>/execute_task.py', 
    '--nlp', user_input
], capture_output=True, text=True)

# 检查输出，如果是确认状态...
# 读取生成的计划书文件
# 展示给用户确认
```

> **注意**：`stp` 是 `structured-task-planning` 的缩写。
