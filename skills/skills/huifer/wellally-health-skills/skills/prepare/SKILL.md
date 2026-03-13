---
name: prepare
description: Hospital visit preparation guide including department recommendations, required documents, and health summary
argument-hint: <就诊目标，如：头疼/心内科/体检 (无参数显示通用指南)>
allowed-tools: Read, Write
schema: prepare/schema.json
---

# Hospital Visit Preparation Skill

Quickly get hospital visit preparation information, including department recommendations, required documents, precautions, and health data summary.

## 核心流程

```
用户输入 -> 识别就诊目标 -> 读取健康数据 -> 生成就诊准备清单 -> 输出健康摘要 -> 提供建议
```

## 步骤 1: 解析用户输入

### Visit Target Recognition

| Input Type | Example | Recommended Department |
|-----------|---------|------------------------|
| Symptom description | Headache, dizziness, vertigo | Neurology |
| Symptom description | Chest pain, chest tightness, palpitations | Cardiology |
| Symptom description | Cough, sputum, difficulty breathing | Respiratory |
| Symptom description | Stomach pain, abdominal pain, diarrhea | Gastroenterology |
| Symptom description | Fever | Fever Clinic or Internal Medicine |
| Department name | Cardiology, Gastroenterology, etc. | Use directly |
| Test item | Physical exam, Ultrasound, CT | Test preparation guide |
| No parameter | - | General preparation guide + health summary |

### Symptom to Department Mapping

| Symptom Keywords | Recommended Department | Notes |
|-----------------|------------------------|-------|
| Headache, dizziness, vertigo | Neurology | Cardiology if with hypertension |
| Chest pain, chest tightness, palpitations | Cardiology | Emergency for urgent cases |
| Cough, sputum, difficulty breathing | Respiratory | |
| Stomach pain, abdominal pain, diarrhea | Gastroenterology | Emergency for acute severe pain |
| Fever | Fever Clinic or Internal Medicine | |
| Rash, itching | Dermatology | |
| Joint pain, lower back pain | Orthopedics or Rheumatology | Orthopedics for trauma |
| Urinary frequency, urgency, pain | Urology | |
| Eye pain, blurred vision | Ophthalmology | |
| Ear pain, hearing loss | ENT | |
| Throat pain, hoarseness | ENT | |
| Breast lump | Breast Surgery | |
| Thyroid nodule | Thyroid Surgery or Endocrinology | |
| Diabetes, high blood sugar | Endocrinology | |
| Hypertension | Cardiology | |
| Childhood illness | Pediatrics | |
| Female gynecological issues | Gynecology | |
| Obstetric checkup | Obstetrics | |
| Mental, emotional, sleep issues | Psychiatry or Psychology | |
| Unexplained physical discomfort | General Practice / Internal Medicine | |

## 步骤 2: 读取健康数据

从系统数据中读取：
- `data/index.json`: 检查记录索引
- `data/health-feeling-logs.json`: 最近症状记录
- `data/allergies.json`: 过敏史 (重点!)
- 慢性病管理数据
- 待复查项目

### Health Summary Includes

1. **Basic Information**: Age, blood type
2. **Allergy History Highlights** (3 items, must be marked!)
3. **Recent Symptoms** (within 7 days)
4. **Current Medications**
5. **Recent Examinations** (within 30 days)
6. **Recent Diagnoses**
7. **Pending Follow-up Items**

## 步骤 3: 生成就诊准备清单

### 必备证件清单

**通用证件:**
- ☐ 身份证/医保卡/社保卡 (必带)
- ☐ 医保卡/就诊卡 (必带)
- ☐ 银行卡或手机支付
- ☐ 既往病历资料 (如有)

**过敏相关** (非常重要!):
- ☐ 过敏史清单 (必带)
- ☐ 过敏急救药物 (如严重过敏)

### Document Preparation Checklist

- ☐ Recent test reports and imaging studies
- ☐ Medication list or currently taking medications
- ☐ Past discharge summaries (if available)
- ☐ Allergy history list (required!)
- ☐ Allergy emergency medications (if carrying!)
- ☐ Family medical history information

## 步骤 4: 输出格式

### 无参数时 (通用准备指南)

```markdown
 医院就诊准备指南

 就诊准备清单
必备证件、资料准备、就诊流程、就诊技巧

您的健康摘要:
基本信息、过敏史重点提示、最近症状、正在用药、最近检查

就诊建议:
根据症状和病史推荐科室

 就诊提示:
就诊时请向医生提供关键信息
```

### 有参数时 (特定目标)

```markdown
 就诊准备指南

就诊信息: 症状/科室/检查项目
推荐科室: XXX

必备证件: [清单]
资料准备: [清单]
注意事项: [事项]

您的健康数据摘要:
[健康摘要]

 就诊提示:
[具体提示]
```

## 执行指令

```
1. 识别就诊目标 (症状/科室/检查/无)
2. 如有症状，匹配推荐科室
3. 读取用户健康数据
4. 生成健康摘要
5. 生成就诊准备清单
6. 根据目标提供科室专项建议
7. 向用户展示完整准备信息
```

## 示例交互

### 示例 1: 无参数 (通用指南+健康摘要)
```
用户: 就诊准备
AI:  医院就诊准备指南

     就诊准备清单
    必备证件、资料准备、就诊流程...

    您的健康摘要:
    基本信息: 年龄45岁，A型血

     过敏史重点提示 (必须主动告知!):
    • 青霉素 - 严重过敏
    • 碘造影剂 - 严重过敏
    • 蜜蜂叮咬 - 过敏性休克

    最近症状、正在用药、最近检查...

    就诊建议: 根据症状推荐科室

     这份健康摘要可以在就诊时向医生展示
```

### 示例 2: 根据症状推荐
```
用户: 就诊准备 头疼
AI:  就诊准备指南

    就诊信息:
    症状: 头疼
    推荐科室: 神经内科

    科室简介:
    神经内科主要诊治脑血管疾病、头痛、
    头晕、失眠、癫痫等神经系统疾病。

    必备证件: [清单]
    资料准备: [清单]
    注意事项: [事项]

    您的健康数据摘要:
    [摘要内容...]

     就诊提示:
    就诊时请向医生提供:
    1. 主要症状: 头疼2天，以胀痛为主
    2. 既往病史: 高血压2级，中危组
    3. 正在用药: 阿司匹林、氨氯地平
    4. 过敏史: 青霉素 (必须告知!)

    建议重点关注:
    • 头疼与高血压的关系
    • 当前用药是否需要调整
```

### 示例 3: 指定科室
```
用户: 就诊准备 心内科
AI:  心内科就诊准备指南

    科室简介:
    心内科主要诊治心脏病、高血压、
    血脂异常、心律失常等疾病。

    常见检查:
    心电图、心脏彩超、动态心电图、冠脉造影

    准备事项:
    • 检查当天穿宽松衣物
    • 24小时动态心电图需提前预约
    • 冠脉造影需空腹，并有人陪同

    您的健康数据摘要: [摘要...]

     就诊提示: [心内科专项提示]
```

## 重要提示

- **过敏史必须第一时间告知医护人员!**
- 就诊时准备做笔记或录音
- 就诊流程可能因医院而异
- 紧急情况请直接拨打急救电话或前往急诊
- 所有健康数据仅保存在本地
