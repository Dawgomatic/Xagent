---
name: interaction
description: Check and manage drug interactions including drug-drug, drug-disease, drug-dosage, and drug-food interactions with 5-level severity classification.
argument-hint: <操作类型(检查/列出/添加/更新/删除/历史) 药物名称>
allowed-tools: Read, Write
schema: interaction/schema.json
---

# 药物相互作用检查技能

检测和管理药物相互作用，包括药物-药物、药物-疾病、药物剂量、药物-食物四种类型的相互作用。

## 核心流程

```
用户输入 -> 解析操作类型 -> [check] 加载用药 -> 执行检测 -> 汇总排序 -> 输出报告
                              -> [list] 读取规则库 -> 格式化显示
                              -> [add] 解析规则 -> 保存
                              -> [update] 查找规则 -> 更新
                              -> [delete] 确认删除
                              -> [history] 显示历史
```

## 步骤 1: 解析操作类型

| Input Keywords | Operation |
|---------------|-----------|
| check | check |
| list | list |
| add | add |
| update | update |
| delete | delete |
| history | history |

## 步骤 2: 检查相互作用 (check)

### 四种检测类型

1. **药物-药物相互作用**：检测当前用药之间的相互作用
2. **药物-疾病冲突**：检测药物与用户疾病史的冲突
3. **药物剂量冲突**：检测剂量是否超过安全范围
4. **药物-食物相互作用**：检测药物与饮食的相互作用

### 五级严重程度分级

| 级别 | 名称 | 代码 | 处理建议 |
|-----|------|------|----------|
| A | 安全 | 1 | 无需特别措施 |
| B | 谨慎使用 | 2 | 注意观察 |
| C | 相对禁忌 | 3 | 权衡获益风险 |
| D | 禁忌 | 4 | 避免合用 |
| X | 绝对禁忌 | 5 | 严格禁止 |

### 检测流程

```
1. 从 data/medications/medications.json 读取当前用药
2. 从 data/profile.json 读取疾病史
3. 从 data/interactions/interaction-db.json 读取相互作用规则
4. 执行四种检测逻辑
5. 按严重程度排序
6. 生成报告
```

## 步骤 3: 保存检查记录

文件路径：`data/interactions/interaction-logs/YYYY-MM/YYYY-MM-DD.json`

```json
{
  "date": "2025-12-31",
  "interactions_detected": {
    "total": 3,
    "by_severity": {
      "X": 1,
      "D": 1,
      "C": 1
    }
  }
}
```

## 步骤 4: 列出相互作用规则 (list)

从 `data/interactions/interaction-db.json` 读取并显示。

## 执行指令

```
1. 解析操作类型
2. [check] 执行四种检测 -> 汇总排序 -> 输出报告 -> 保存
3. [list] 读取规则库 -> 统计显示
4. [add] 解析规则 -> 验证 -> 保存
5. [update] 查找规则 -> 更新字段 -> 保存
6. [delete] 查找规则 -> 确认 -> 删除
7. [history] 读取历史 -> 格式化显示
```

## 示例交互

### 检查相互作用
```
用户: 检查药物相互作用

输出:
 药物相互作用检查
检测到 3 个相互作用：
 绝对禁忌：华法林 + 阿司匹林
 禁忌：辛伐他汀 + 氨氯地平
 相对禁忌：华法林 + 富含维生素K的食物
```

### 查看规则库
```
用户: 列出相互作用规则

输出:
 药物相互作用规则库
总规则数：80 条
 绝对禁忌：5 条
...
```

### 添加规则
```
用户: 添加相互作用 阿司匹林 华法林 X 显著增加出血风险

保存新规则
```
