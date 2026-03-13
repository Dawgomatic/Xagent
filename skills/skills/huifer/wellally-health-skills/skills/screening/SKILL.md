---
name: screening
description: Manage gynecological cancer screening including HPV testing, TCT (Pap smear), co-testing, and tumor marker tracking. Use for cervical cancer screening, ovarian cancer monitoring, and gynecological health records.
argument-hint: <操作类型 筛查结果，如：hpv negative / tct NILM / co-testing hpv阴性 tct正常>
allowed-tools: Read, Write
schema: screening/schema.json
---

# 妇科癌症筛查追踪技能

宫颈癌、卵巢癌、子宫内膜癌筛查计划管理和结果追踪。

## 核心流程

```
用户输入 → 识别操作类型 → [hpv/tct] 解析结果 → 风险评估 → 保存记录
                              ↓
                         [co-testing] 解析联合结果 → 综合评估 → 保存
                              ↓
                         [marker] 解析肿瘤标志物 → 趋势分析 → 保存
                              ↓
                         [status/next] 读取数据 → 显示报告
```

## 步骤 1: 解析用户输入

### 操作类型识别

| Input Keywords | Operation Type |
|---------------|----------------|
| hpv, HPV | hpv - Record HPV test |
| tct, TCT | tct - Record TCT test |
| co-testing | co-testing - Joint screening |
| marker | marker - Tumor marker |
| abnormal | abnormal - Abnormal result follow-up |
| status | status - View screening status |
| next | next - Next screening reminder |

## 步骤 2: 参数解析规则

### HPV结果识别

| User Input | Standard Result |
|-----------|---------------|
| negative | negative |
| positive | positive |
| 16, 18, 31, 33, 52, 58 | HPV type |

### HPV型别分类

| 风险等级 | HPV型别 |
|---------|---------|
| 高危（最高危） | 16, 18 |
| 高危（其他） | 31, 33, 35, 39, 45, 51, 52, 56, 58, 59 |
| 低危 | 6, 11, 40, 42, 43, 44, 54, 61, 70, 72, 81 |

### TCT结果识别（Bethesda系统）

| User Input | Standard Result | Risk |
|-----------|---------------|------|
| NILM | NILM | Normal |
| ASC-US | ASC-US | Low |
| ASC-H | ASC-H | Medium-High |
| LSIL | LSIL | Low-Medium |
| HSIL | HSIL | High |
| AGC | AGC | Medium-High |
| cancer | Cancer | Very High |

### 肿瘤标志物识别

| 标志物 | 相关癌症 | 正常值 |
|--------|---------|--------|
| CA125 | 卵巢癌、子宫内膜癌 | <35 U/mL |
| CA19-9 | 卵巢癌、子宫内膜癌 | <37 U/mL |
| CEA | 子宫内膜癌 | <5 ng/mL |
| AFP | 卵黄囊瘤 | <10 ng/mL |

## 步骤 3: 风险评估

### HPV阴性
```
 HPV阴性
风险评估：低
管理建议：继续常规筛查
```

### HPV 16/18阳性（最高危）
```
 HPV 16/18阳性（最高危）
风险评估：高
立即行动：立即进行阴道镜检查
```

### 其他高危HPV阳性
```
 高危HPV阳性
风险评估：中-高
管理建议：1年后复查 或 立即阴道镜
```

### TCT异常结果解读

#### ASC-US
```
 TCT结果：ASC-US
风险评估：CIN 2+风险约5-10%
管理方案：反射HPV检测
```

#### LSIL
```
 TCT结果：LSIL
风险评估：CIN 2+风险约15-20%
管理方案：1年后复查TCT+HPV
```

#### HSIL
```
 TCT结果：HSIL
风险评估：CIN 2+风险>50%
立即行动：立即阴道镜检查+活检
```

### 联合筛查结果管理

| HPV | TCT | 风险 | 管理 |
|-----|-----|------|------|
| 阴性 | NILM | 极低 | 5年后复查 |
| 阳性16/18 | 任何TCT | 高 | 立即阴道镜 |
| 阳性其他 | NILM | 低-中 | 1年后复查 |
| 阳性其他 | ASC-US | 中 | 阴道镜或1年后复查 |
| 阴性 | ASC-US | 低 | 3年后复查 |

## 步骤 4: 生成 JSON

### HPV检测记录

```json
{
  "last_hpv": "2025-01-15",
  "hpv_result": "positive",
  "hpv_type": "16",
  "hpv_risk_level": "high",
  "next_screening_type": "colposcopy",
  "days_until_next": 0
}
```

### TCT检测记录

```json
{
  "last_tct": "2025-01-15",
  "tct_result": "ASC-US",
  "tct_result_full": "非典型鳞状细胞，意义不明确",
  "tct_bethesda_category": "ASC-US"
}
```

### 肿瘤标志物记录

```json
{
  "CA125": {
    "current_value": 15.5,
    "reference_range": "<35",
    "unit": "U/mL",
    "last_checked": "2025-06-20",
    "classification": "normal",
    "trend": "stable"
  }
}
```

## 步骤 5: 保存数据

1. 读取 `data/screening-tracker.json`
2. 更新对应记录段
3. 写回文件

## 筛查策略

### 年龄分层筛查

| 年龄 | 筛查策略 | 筛查间隔 |
|-----|---------|---------|
| 21-29岁 | TCT单独 | 每3年 |
| 30-65岁 | HPV+TCT联合 | 每5年 |
| 或TCT单独 | | 每3年 |

### 高风险人群

| 人群类型 | 筛查频率 |
|---------|---------|
| HIV阳性 | 每年1次 |
| 免疫抑制 | 每年1次 |
| 宫颈癌病史 | 每年1次 |
| DES暴露者 | 每年1次 |

更多示例参见 [examples.md](examples.md)。
