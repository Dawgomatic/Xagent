# Health Goals Schema

健康目标与习惯管理的完整数据结构定义。

## Schema 文件

完整的 JSON Schema 定义：[schema.json](schema.json)

## 字段速查

### 目标 (goals)

| 字段 | 类型 | 说明 |
|-----|------|-----|
| `goal_id` | string | 目标唯一标识符 (goal_YYYYMMDD_HHMMSS) |
| `goal_type` | enum | 目标类型 |
| `title` | string | 目标标题 |
| `description` | string | 目标描述 |
| `start_date` | string | 开始日期 (YYYY-MM-DD) |
| `target_date` | string | 目标日期 (YYYY-MM-DD) |
| `status` | enum | active/paused/completed/archived |
| `target_value` | object | 目标值 {value, unit} |
| `current_value` | object | 当前值 {value, unit} |
| `progress_percentage` | number | 进度百分比 (0-100) |

### 目标类型 (goal_type)

`weight_loss` (减重) | `weight_gain` (增重) | `exercise` (运动) | `diet` (饮食) | `sleep` (睡眠) | `health_metric` (健康指标) | `custom` (自定义)

### SMART 验证 (smart_validation)

| 字段 | 类型 | 说明 |
|-----|------|-----|
| `specific` | boolean | 具体 |
| `measurable` | boolean | 可衡量 |
| `achievable` | boolean | 可实现 |
| `relevant` | boolean | 相关 |
| `time_bound` | boolean | 有时限 |

### 习惯 (habits)

| 字段 | 类型 | 说明 |
|-----|------|-----|
| `habit_id` | string | 习惯唯一标识符 |
| `name` | string | 习惯名称 |
| `habit_type` | enum | daily/weekly/trigger_based |
| `trigger` | string | 触发条件 |
| `streak_count` | int | 连续打卡天数 |
| `longest_streak` | int | 最长连续天数 |
| `completion_rate` | number | 完成率 (0-100) |

### 成就 (achievements)

| 成就ID | 标题 | 描述 | 图标 |
|-------|------|------|------|
| `first_goal` | 首次目标 | 完成第一个健康目标 |  |
| `streak_7` | 连续7天 | 任意习惯连续7天打卡 |  |
| `streak_21` | 连续21天 | 任意习惯连续21天打卡 |  |
| `streak_30` | 连续30天 | 任意习惯连续30天打卡 |  |
| `halfway` | 半程达成 | 任意目标完成50% |  |
| `goal_complete` | 目标达成 | 完成一个健康目标 |  |
| `early_complete` | 提前完成 | 提前完成目标 |  |
| `exceed_goal` | 超额完成 | 超额完成目标 |  |

## 数据存储

- 位置: `data/health-goals-tracker.json`
- 格式: JSON 对象
- 模式: 更新
