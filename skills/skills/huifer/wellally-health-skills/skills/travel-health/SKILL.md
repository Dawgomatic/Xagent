---
name: travel-health
description: Manage travel health data including risk assessment, vaccination records, travel kit management, medication tracking, insurance information, emergency contacts, and checkup reminders. Use for pre-trip planning, health preparation, and travel health safety.
argument-hint: <操作类型 旅行信息，如：plan Thailand 2025-08-01 for 14 days tourism>
allowed-tools: Read, Write
schema: travel-health/schema.json
---

# Travel Health Management Skill

Manage travel health data, plan travel health preparation, assess destination health risks, and manage vaccination and travel kits.

## Medical Safety Disclaimer

All health advice and information provided by this system is for reference only and cannot replace professional medical advice.

**Cannot Do:**
- Do not provide specific medical prescriptions or diagnoses
- Vaccination and medication regimens must be determined by doctors
- Destination health risk data may have latency

**Must Do:**
- Must consult a doctor 4-6 weeks before travel
- Vaccination must be determined by doctors based on individual health status

## 数据来源

- 世界卫生组织(WHO): https://www.who.int/ith
- 美国疾控中心(CDC): https://www.cdc.gov/travel

## 核心流程

```
用户输入 → 识别操作类型 → [plan] 解析目的地/日期 → 风险评估 → 生成建议
                              ↓
                         [vaccine] 更新疫苗记录 → 检查接种计划
                              ↓
                         [kit] 管理药箱物品清单
                              ↓
                         [status] 读取数据 → 显示准备状态
                              ↓
                         [risk] 评估目的地健康风险
```

## 步骤 1: 解析用户输入

### 操作类型识别

| Input Keywords | Operation Type |
|---------------|----------------|
| plan, 规划 | plan - Plan travel health preparation |
| vaccine, 疫苗 | vaccine - Vaccination management |
| kit, 药箱 | kit - Travel kit management |
| medication, 用药 | medication - Medication management |
| insurance, 保险 | insurance - Insurance information |
| emergency, 紧急 | emergency - Emergency contacts |
| status, 状态 | status - Preparation status |
| risk, 风险 | risk - Risk assessment |
| check, 检查 | check - Health check |
| card, 卡片 | card - Emergency card |
| alert, 预警 | alert - Epidemic alert |

## 步骤 2: 目的地风险评估

### 风险等级

| 等级 | 描述 | 预防措施 |
|-----|------|---------|
|  低风险 | 常规预防措施 | 基础疫苗接种 |
|  中等风险 | 需要特别注意 | 特定疫苗 + 注意事项 |
|  高风险 | 需要严格预防措施 | 特定疫苗 + 预防性用药 |
|  极高风险 | 建议推迟旅行 | 特殊预防 + 专业咨询 |

### 常见目的地风险

| 地区 | 主要风险 | 建议疫苗 |
|-----|---------|---------|
| 东南亚 | 甲肝、伤寒、登革热 | 甲肝疫苗、伤寒疫苗 |
| 非洲 | 黄热病、疟疾、霍乱 | 黄热病疫苗、疟疾预防 |
| 南亚 | 甲肝、伤寒、霍乱 | 甲肝疫苗、伤寒疫苗、霍乱疫苗 |

## 步骤 3: 生成 JSON

### 旅行计划数据结构

```json
{
  "trip_id": "trip_20250801_seasia",
  "destination": ["Thailand", "Vietnam", "Cambodia"],
  "start_date": "2025-08-01",
  "end_date": "2025-08-15",
  "duration_days": 14,
  "travel_type": "tourism",
  "risk_assessment": {
    "overall_risk": "medium",
    "vaccinations_needed": ["hepatitis_a", "typhoid"],
    "medications_recommended": ["anti_malarial"],
    "health_advisories": ["food_safety", "water_safety"]
  }
}
```

### 疫苗记录数据结构

```json
{
  "vaccine_id": "vaccine_001",
  "vaccine_type": "hepatitis_a",
  "date_administered": "2025-06-15",
  "status": "completed",
  "next_dose_due": null,
  "notes": ""
}
```

### 旅行药箱数据结构

```json
{
  "kit_items": [
    {
      "name": "antidiarrheal",
      "category": "medication",
      "quantity": 2,
      "packed": false
    }
  ]
}
```

## 步骤 4: 保存数据

1. 读取 `data/travel-health-tracker.json`
2. 更新对应记录段
3. 写回文件

## 旅行前准备时间表

| 时间 | 准备事项 |
|-----|---------|
| 出发前6-8周 | 规划旅行健康、咨询医生、开始疫苗接种 |
| 出发前4-6周 | 完成疫苗接种、准备旅行药箱 |
| 出发前2-4周 | 购买保险、设置紧急联系人、生成紧急卡片 |
| 出发前1周 | 最终健康检查、确认所有准备就绪 |

## 常用旅行药箱物品

### 药品类
- 止泻药
- 退烧止痛药
- 抗过敏药
- 晕车药
- 抗酸药

### 医疗用品
- 创可贴
- 消毒纸巾
- 绷带
- 医用胶带
- 体温计

### 防护用品
- 防晒霜
- 驱蚊剂
- 避蚊手环
| 输入 | 标志物 |
|-----|--------|
| ca125 | CA125 |
| ca19-9 | CA19-9 |
| cea | CEA |
| afp | AFP |

更多示例参见 [examples.md](examples.md)。
