# screening 技能示例

## 一、HPV检测

### 示例 1: HPV阴性
```
用户: /screening hpv negative

保存:
{
  "last_hpv": "2025-01-15",
  "hpv_result": "negative",
  "hpv_risk_level": "none",
  "next_screening": "2030-01-15",
  "days_until_next": 1825
}

输出:
 HPV检测记录已更新
风险评估：低
下次筛查：2030年1月15日（还有1825天）
```

### 示例 2: HPV 16阳性
```
用户: /screening hpv positive 16

保存:
{
  "hpv_result": "positive",
  "hpv_type": ["16"],
  "hpv_risk_level": "high",
  "next_screening_type": "colposcopy",
  "days_until_next": 0
}

输出:
 HPV 16阳性（最高危）
立即行动：立即进行阴道镜检查
```

### 示例 3: HPV 52 58阳性
```
用户: /screening hpv positive 52 58

保存:
{
  "hpv_type": ["52", "58"],
  "hpv_risk_level": "moderate"
}
```

## 二、TCT检测

### 示例 4: TCT正常
```
用户: /screening tct NILM

保存:
{
  "tct_result": "NILM",
  "tct_result_full": "阴性，上皮内病变或恶性病变",
  "tct_bethesda_category": "NILM"
}
```

### 示例 5: ASC-US
```
用户: /screening tct ASC-US

保存:
{
  "tct_result": "ASC-US",
  "tct_result_full": "非典型鳞状细胞，意义不明确"
}

输出:
 TCT结果：ASC-US
管理建议：反射HPV检测
```

### 示例 6: HSIL
```
用户: /screening tct HSIL

保存:
{
  "tct_result": "HSIL"
}

输出:
 TCT结果：HSIL
立即行动：立即进行阴道镜检查+活检
```

## 三、联合筛查

### 示例 7: 双阴性
```
用户: /screening co-testing negative NILM

保存:
{
  "hpv_result": "negative",
  "tct_result": "NILM",
  "co_testing_result": "negative_NILM",
  "co_testing_interpretation": "极低风险"
}

输出:
 联合筛查结果：HPV阴性 + TCT正常
宫颈癌风险：极低
下次筛查：2030年1月15日（5年）
```

### 示例 8: HPV阳性TCT正常
```
用户: /screening co-testing hpv阳性 tct正常

保存:
{
  "hpv_result": "positive",
  "tct_result": "NILM"
}

输出:
风险评估：高
即使TCT正常，HPV阳性也需要阴道镜检查
```

## 四、肿瘤标志物

### 示例 9: CA125正常
```
用户: /screening marker ca125 15.5

保存:
{
  "CA125": {
    "current_value": 15.5,
    "reference_range": "<35",
    "classification": "normal",
    "trend": "stable"
  }
}
```

### 示例 10: CA125升高
```
用户: /screening marker ca125 80

保存:
{
  "current_value": 80,
  "classification": "significantly_elevated"
}

输出:
 CA125显著升高
建议：立即咨询妇科医生
```

### 示例 11: 多个标志物
```
用户: /screening marker ca125 15.5 ca19-9 22.0 cea 3.2

保存多个标志物结果
```

## 五、异常结果随访

### 示例 12: 记录随访
```
用户: /screening abnormal colposcopy scheduled 2025-02-01

保存:
{
  "follow_up": {
    "type": "colposcopy",
    "scheduled_date": "2025-02-01",
    "status": "pending"
  }
}
```

### 示例 13: 随访完成
```
用户: /screening abnormal colposcopy normal

保存:
{
  "follow_up": {
    "type": "colposcopy",
    "result": "normal",
    "status": "resolved"
  }
}
```

## 六、查看状态

### 示例 14: 查看筛查状态
```
用户: /screening status

输出:
 妇科癌症筛查状态

宫颈癌筛查：
━━━━━━━━━━━━━━━━━━━━━━━━━━
筛查策略：联合筛查（HPV+TCT）

最近检查：
━━━━━━━━━━━━━━━━━━━━━━━━━━
HPV检测（2025-01-15）：阴性 
TCT检测（2025-01-15）：NILM（正常）

综合评估：
━━━━━━━━━━━━━━━━━━━━━━━━━━
宫颈癌风险：极低

下次筛查：2030年1月15日（还有1825天）

肿瘤标志物：
━━━━━━━━━━━━━━━━━━━━━━━━━━
CA125：15.5 U/mL（正常）
CA19-9：22.0 U/mL（正常）
CEA：未检测 
AFP：未检测 
```

## 七、下次筛查提醒

### 示例 15: 查看下次筛查
```
用户: /screening next

输出:
 下次筛查提醒

检查类型：联合筛查（HPV+TCT）
预约日期：2030年1月15日（周一）
还有1825天（5年）

检查项目：
• HPV检测（高危型）
• TCT（液基薄层细胞检测）

准备事项：
• 避免月经期（最好在月经干净后3-7天）
• 检查前24-48小时避免性生活
• 检查前24-48小时避免阴道冲洗
• 检查前24-48小时避免使用阴道药物
```
