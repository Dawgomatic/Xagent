---
name: surgery
description: Personal surgery history recording with implant information tracking and structured data extraction
argument-hint: <手术描述，如：去年8月15日腹腔镜下胆囊切除术 因为慢性结石性胆囊炎>
allowed-tools: Read, Write
schema: surgery/schema.json
---

# Personal Surgery History Record Skill

Record personal surgery history, extracting structured surgical information from natural language descriptions.

## 核心流程

```
用户输入 -> 解析手术信息 -> 询问植入物 -> 确认信息 -> 生成JSON -> 保存数据
```

## 步骤 1: 解析用户输入

### 必需字段提取

| Field | Extraction Rule |
|-------|----------------|
| Surgery full name | Medical terminology, e.g., "Laparoscopic cholecystectomy" |
| Surgery alias | Common description, e.g., "Gallbladder minimally invasive surgery" |
| Surgery reason | Preoperative diagnosis, e.g., "Chronic calculous cholecystitis" |
| Surgery date | YYYY-MM-DD format |
| Surgery site | Body part, e.g., "abdomen" |

### 可选字段提取

| Field | Description |
|-------|-------------|
| Surgery type | elective/emergency/day_surgery |
| Anesthesia type | General anesthesia/local anesthesia/spinal anesthesia, etc. |
| Surgery duration | Minutes |
| Intraoperative blood loss | Milliliters |
| Lead surgeon | Doctor's name |
| Surgery hospital | Hospital name |
| Hospitalization days | Days |

### 手术类型分类

| 类型 | 说明 |
|------|------|
| elective | 可以提前安排的手术，非紧急情况 |
| emergency | 需要立即进行的紧急手术 |
| day_surgery | 当天住院、当天出院的手术 |
| diagnostic | 主要用于明确诊断的手术 |
| therapeutic | 主要用于治疗的手术 |
| palliative | 减轻症状但不能治愈的手术 |
| reconstructive | 重建或修复功能的手术 |

## 步骤 2: 询问植入物信息

**必须询问用户:**

```
 植入物信息确认

本次手术中是否有植入以下任何医疗材料？
- 人工关节/植入物
- 支架/导管
- 金属内固定物（钢板、螺钉、钢针等）
- 人工瓣膜/起搏器
- 疝修补片/补片
- 其他植入物

A. 有植入物
B. 无植入物
```

如果选择"有植入物"，则询问详细信息：
- 植入物名称
- 植入物型号/规格
- 植入部位
- 植入物数量
- 预计取出时间

## 步骤 3: 生成 JSON

### 手术数据结构

```json
{
  "id": "surgery_20240815_001",
  "basic_info": {
    "surgery_name": "腹腔镜下胆囊切除术",
    "surgery_alias": "胆囊微创手术",
    "surgery_date": "2024-08-15",
    "preoperative_diagnosis": "慢性结石性胆囊炎",
    "body_part": "腹部",
    "surgery_type": "elective",
    "anesthesia_type": "全身麻醉",
    "duration_minutes": 90,
    "blood_loss_ml": 50,
    "surgeon": "张医生",
    "hospital": "某某医院",
    "hospitalization_days": 3
  },
  "implants": {
    "has_implants": false,
    "implants_list": []
  },
  "postoperative_info": {
    "complications": null,
    "recovery_status": "良好",
    "follow_up_plan": null
  },
  "notes": "",
  "created_at": "2024-08-15",
  "original_input": "用户原始输入描述"
}
```

### 植入物数据结构

```json
{
  "has_implants": true,
  "implants_list": [
    {
      "implant_name": "钛合金钢板",
      "model": "××品牌×型号",
      "location": "右胫骨中段",
      "quantity": 1,
      "removal_plan": "12个月后取出",
      "implant_date": "2024-08-15"
    }
  ]
}
```

完整 Schema 定义参见 [schema.json](schema.json)。

## 步骤 4: 保存数据

1. 生成唯一ID: `surgery_YYYYMMDD_XXX`
2. 保存到 `data/手术记录/YYYY-MM/YYYY-MM-DD_手术名称.json`
3. 更新 `data/index.json`

## 步骤 5: 输出确认

```
 手术记录已保存

基本信息:
━━━━━━━━━━━━━━━━━━━━━━━━━━
手术名称: 腹腔镜下胆囊切除术（胆囊微创手术）
手术日期: 2024-08-15
术前诊断: 慢性结石性胆囊炎
手术部位: 腹部
手术类型: 择期手术
麻醉方式: 全身麻醉
手术时长: 90分钟
术中出血: 50ml

植入物信息:
━━━━━━━━━━━━━━━━━━━━━━━━━━
无植入物

数据已保存至:
data/手术记录/2024-08/2024-08-15_腹腔镜下胆囊切除术.json
```

## 常见手术示例

### 示例 1: 胆囊手术
```
用户: 去年8月15日做了腹腔镜下胆囊切除术，因为慢性结石性胆囊炎
AI: [解析信息并确认植入物...]
```

### 示例 2: 骨折手术
```
用户: 2024年3月10日右腿胫骨骨折内固定术，车祸
AI: [解析信息并询问植入物...]
    检测到骨折内固定术，需要确认植入物信息...
```

### 示例 3: 冠脉支架
```
用户: 2023年12月做了冠状动脉支架植入术，心绞痛
AI: [解析信息并确认支架信息...]
    检测到支架植入，需要确认支架详情...
```

## 执行指令

```
1. 解析自然语言描述，提取手术信息
2. 识别手术名称、原因、日期、部位
3. 询问并确认植入物信息 (重要!)
4. 补充可选信息 (如缺失)
5. 生成符合 schema.json 的 JSON
6. 保存到相应文件路径
7. 更新全局索引
8. 向用户确认保存成功，并显示摘要
```

## 重要提示

- 植入物信息特别重要，必须确认
- 日期格式统一转换为 YYYY-MM-DD
- 尽可能完整记录手术信息
- 保持数据结构化，便于后续查询和分析
- 所有数据仅保存在本地

## 手术记录查询

手术记录可以通过以下方式查询：
- 按时间范围查询
- 按手术部位查询
- 按手术名称查询
