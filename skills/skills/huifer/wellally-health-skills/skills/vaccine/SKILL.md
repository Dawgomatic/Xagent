---
name: vaccine
description: Vaccination record management with schedule tracking, safety checks, and adverse reaction monitoring
argument-hint: <操作类型，如：add 乙肝疫苗 第一针/record 乙肝疫苗 第2针/schedule/due/check 乙肝疫苗>
allowed-tools: Read, Write
schema: vaccine/schema.json
---

# Vaccination Record Management Skill

Manage vaccination records and schedules, support multi-dose vaccine tracking, vaccination plan management, adverse reaction recording, and safety checks.

## 核心流程

```
用户输入 -> 识别操作类型 -> 解析信息 -> 接种前安全检查 -> 生成计划/记录 -> 保存数据
```

## 步骤 1: 解析用户输入

### 操作类型识别

| Input Keywords | Operation Type |
|---------------|----------------|
| add, 添加 | Add vaccination plan |
| record, 记录 | Record actual vaccination |
| schedule, 计划 | View vaccination schedule |
| due, 待种 | View pending vaccinations |
| history, 历史 | View vaccination history |
| status, 状态 | View vaccination statistics |
| check, 检查 | Pre-vaccination safety check |

### 疫苗名称识别

| 疫苗 | 别名 |
|------|------|
| 乙肝疫苗 | HepB, 乙型肝炎疫苗 |
| 流感疫苗 | Flu vaccine, 流行性感冒疫苗 |
| HPV疫苗 | 宫颈癌疫苗, 人乳头瘤病毒疫苗 |
| COVID-19疫苗 | 新冠疫苗, 冠状病毒疫苗 |
| 百白破疫苗 | DPT |
| 麻腮风疫苗 | MMR |
| 脊髓灰质炎疫苗 | 脊灰疫苗 |

### 剂次识别

| 用户输入 | 标准化 |
|---------|--------|
| 第1针、第一针、第1剂 | dose_number: 1 |
| 第2针、第二针、第2剂 | dose_number: 2 |
| 第3针、第三针、第3剂 | dose_number: 3 |

### 接种程序识别

| 用户输入 | 标准化 | 总剂次 |
|---------|--------|-------|
| 0-1-6, 016程序 | 0-1-6 | 3剂 |
| 0-2-6, 026程序 | 0-2-6 | 3剂 |
| 单次, 1次 | single | 1剂 |

## 步骤 2: 接种前安全检查

### 安全检查项目

#### 1. 过敏检查

```javascript
// 伪代码示例
function checkVaccineAllergies(vaccine) {
  const allergies = loadAllergies('data/allergies.json');
  const warnings = [];

  for (const allergy of allergies.allergies) {
    if (allergy.current_status.status !== 'active') continue;

    const isContraindication = vaccine.contraindications.some(c =>
      c.type === 'allergy' && c.allergen === allergy.allergen.name
    );

    if (isContraindication) {
      warnings.push({
        allergen: allergy.allergen.name,
        severity: allergy.severity.level,
        recommendation: getRecommendation(allergy.severity.level)
      });
    }
  }

  return warnings;
}
```

#### 2. 年龄适宜性检查

```javascript
function checkAgeAppropriateness(vaccine, birthDate) {
  const age = calculateAge(birthDate);
  const recommendation = vaccine.age_recommendations;

  if (age < recommendation.min_age) {
    return {
      appropriate: false,
      reason: `年龄不足，建议${recommendation.min_age}后再接种`
    };
  }

  return {
    appropriate: true,
    recommended_age: recommendation.recommended_age
  };
}
```

#### 3. 药物相互作用检查

```javascript
function checkVaccineInteractions(vaccine) {
  const medications = loadMedications();
  const interactions = [];

  for (const vaccineInteraction of vaccine.interactions) {
    const matchingMeds = medications.filter(med =>
      med.active && med.category === vaccineInteraction.drug_category
    );

    if (matchingMeds.length > 0) {
      interactions.push({
        drugs: matchingMeds.map(m => m.name),
        interaction: vaccineInteraction
      });
    }
  }

  return interactions;
}
```

## 步骤 3: 生成接种计划

### 接种计划生成算法

```javascript
function generateVaccineSchedule(vaccine, firstDoseDate) {
  const scheduleTypes = {
    '0-1-6': [
      { dose: 1, offset: 0, unit: 'months' },
      { dose: 2, offset: 1, unit: 'months' },
      { dose: 3, offset: 6, unit: 'months' }
    ],
    '0-2-6': [
      { dose: 1, offset: 0, unit: 'months' },
      { dose: 2, offset: 2, unit: 'months' },
      { dose: 3, offset: 6, unit: 'months' }
    ],
    'annual': [
      { dose: 1, offset: 1, unit: 'years' }
    ],
    'single': [
      { dose: 1, offset: 0, unit: 'days' }
    ]
  };

  const pattern = scheduleTypes[vaccine.standard_schedule];
  const schedule = [];

  for (const doseInfo of pattern) {
    const scheduledDate = addOffset(firstDoseDate, doseInfo.offset, doseInfo.unit);
    const isFirstDose = doseInfo.dose === 1;

    schedule.push({
      dose_number: doseInfo.dose,
      scheduled_date: formatDate(scheduledDate),
      administered_date: isFirstDose && firstDoseDate <= new Date() ? formatDate(firstDoseDate) : null,
      status: isFirstDose && firstDoseDate <= new Date() ? 'completed' : 'scheduled'
    });
  }

  return schedule;
}
```

## 步骤 4: 生成 JSON

### 疫苗接种数据结构

```json
{
  "id": "vax_20251231123456789",
  "vaccine_info": {
    "name": "乙型肝炎疫苗",
    "type": "recombinant",
    "manufacturer": "北京生物制品研究所",
    "batch_number": "202512001",
    "dose_volume": {
      "value": 0.5,
      "unit": "ml"
    },
    "route": "intramuscular"
  },
  "series_info": {
    "is_series": true,
    "series_type": "primary",
    "total_doses": 3,
    "current_dose": 2,
    "schedule_type": "0-1-6"
  },
  "doses": [
    {
      "dose_number": 1,
      "scheduled_date": "2025-11-15",
      "administered_date": "2025-11-15",
      "site": "left_arm",
      "facility": "社区卫生服务中心",
      "status": "completed"
    },
    {
      "dose_number": 2,
      "scheduled_date": "2025-12-15",
      "administered_date": "2025-12-16",
      "site": "right_arm",
      "status": "completed"
    },
    {
      "dose_number": 3,
      "scheduled_date": "2026-05-15",
      "administered_date": null,
      "status": "scheduled"
    }
  ],
  "adverse_reactions": [],
  "safety_checks": {
    "allergy_warnings": [],
    "drug_interactions": [],
    "age_appropriate": true,
    "contraindications": []
  },
  "status": {
    "series_status": "in_progress",
    "completion_percentage": 66.7,
    "next_dose_due": "2026-05-15",
    "is_overdue": false
  }
}
```

完整 Schema 定义参见 [schema.json](schema.json)。

## 步骤 5: 保存数据

1. 生成唯一ID: `vax_YYYYMMDDHHmmssSSS`
2. 保存到 `data/vaccinations.json`
3. 更新统计信息

## 安全警示级别

| 严重程度 | 建议 |
|---------|------|
| mild | 可接种，需观察 |
| moderate | 谨慎接种，建议咨询医生 |
| severe | 不建议接种，或咨询专科医生 |
| anaphylaxis | 绝对禁忌，禁止接种 |

## 执行指令

```
1. 识别操作类型
2. 解析输入参数，提取疫苗信息
3. 从疫苗数据库匹配疫苗信息
4. 执行接种前安全检查:
   - 过敏检查
   - 年龄适宜性检查
   - 药物相互作用检查
   - 禁忌症检查
5. 生成接种计划
6. 生成符合 schema.json 的 JSON
7. 保存到数据文件
8. 向用户确认保存成功，并显示摘要和建议
```

## 示例交互

### 示例 1: 添加乙肝疫苗接种计划
```
用户: 添加疫苗 乙肝疫苗 0-1-6程序 第一针已打昨天
AI:  疫苗接种前安全检查

     过敏史检查:  通过
     年龄适宜性检查:  通过
     当前健康状况检查:  通过
     药物相互作用检查:  无影响
     疫苗接种史检查:  无既往记录

     综合评估: 可以接种

     疫苗接种计划已添加

    乙肝疫苗 0-1-6月程序:
    第1针: 2025-01-07  已接种
    第2针: 2025-02-07 (计划中)
    第3针: 2025-07-07 (计划中)

    进度: 1/3 (33.3%)

     提示: 第2针可在2025-01-24至2025-03-07之间完成
```

### 示例 2: 查看待接种疫苗
```
用户: 待接种疫苗
AI:  待接种提醒

     紧急（已逾期）
    1. 乙肝疫苗第3针
       逾期: 11天（应种: 2025-12-20）
        建议尽快补种

     近期待种（30天内）
    1. HPV疫苗第2针 - 还有15天
    2. 流感疫苗 - 还有30天

    行动建议:
    • 立即联系接种点补种逾期疫苗
    • 为即将到期的疫苗预约接种时间
```

### 示例 3: 接种前安全检查
```
用户: 疫苗检查 乙肝疫苗
AI:  疫苗接种前安全检查

    疫苗: 乙型肝炎疫苗（重组）

     过敏史检查:  通过
       检查结果: 无相关过敏史

     年龄适宜性检查:  通过
       当前年龄: 35岁
       推荐年龄: 任何年龄均可接种

     当前健康状况检查:  通过
       无发热或急性疾病

     药物相互作用检查:  发现1项
       当前用药: 环孢素 100mg 每日2次
       影响: 可能降低疫苗免疫效果
       建议: 接种后2-3个月检测抗体滴度

    ━━━━━━━━━━━━━━━━━━━━━━━━━
    综合评估:  可以接种
    ━━━━━━━━━━━━━━━━━━━━━━━━━

    注意事项:
    • 接种后留观30分钟
    • 如出现不良反应，及时记录
    • 建议接种后2个月检测抗体

    是否继续添加疫苗计划？
    A. 继续添加
    B. 取消
```

## 重要提示

- 本系统仅供个人疫苗接种记录，不能替代专业医疗建议
- 接种前请咨询医生或接种点工作人员
- 如有严重过敏史，必须告知接种人员
- 接种后留观30分钟
- 所有数据仅保存在本地
