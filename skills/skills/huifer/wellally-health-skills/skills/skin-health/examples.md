# skin-health 技能示例

## 一、皮肤问题记录

### 示例 1: 痤疮记录
```
用户: /skin concern acne forehead 额头有痤疮，中度严重

保存:
{
  "issue_type": "acne",
  "severity": "moderate",
  "location": "face",
  "specific_areas": ["forehead"],
  "description": "额头有痤疮，中度严重"
}
```

### 示例 2: 湿疹记录
```
用户: /skin concern eczema手臂内侧湿疹，瘙痒明显

保存:
{
  "issue_type": "eczema",
  "severity": "moderate",
  "location": "body",
  "specific_areas": ["arms"],
  "description": "手臂内侧湿疹，瘙痒明显"
}
```

### 示例 3: 色斑记录
```
用户: /skin concern pigmentation脸颊色斑，日晒后加重

保存:
{
  "issue_type": "pigmentation",
  "location": "face",
  "specific_areas": ["cheeks"],
  "triggers": ["sun_exposure"]
}
```

## 二、痣的监测

### 示例 4: 正常痣记录
```
用户: /skin mole back 4mm 棕色，扁平，边缘规则

保存:
{
  "location": "back",
  "size_mm": 4,
  "color": "brown",
  "shape": "flat",
  "abcde_assessment": {
    "asymmetry": false,
    "border": false,
    "color": false,
    "diameter": false,
    "evolution": false
  },
  "risk_level": "low"
}
```

### 示例 5: 可疑痣记录
```
用户: /skin mole face 3mm 多个颜色混合，建议检查

保存:
{
  "location": "face",
  "size_mm": 3,
  "color": "mixed",
  "abcde_assessment": {
    "color": true
  },
  "risk_level": "moderate",
  "needs_followup": true
}
```

### 示例 6: 不规则痣
```
用户: /skin mole arm 6mm 黑色，轻微隆起，需要注意

保存:
{
  "location": "arm",
  "size_mm": 6,
  "shape": "raised",
  "abcde_assessment": {
    "diameter": true
  },
  "risk_level": "moderate"
}
```

## 三、护肤程序

### 示例 7: 早晨护肤
```
用户: /skin routine morning cleanser toner moisturizer spf30

保存:
{
  "routine_type": "morning",
  "steps": ["cleanser", "toner", "moisturizer", "spf30"]
}
```

### 示例 8: 晚间护肤
```
用户: /skin routine evening cleanser serum moisturizer

保存:
{
  "routine_type": "evening",
  "steps": ["cleanser", "serum", "moisturizer"]
}
```

### 示例 9: 每周护理
```
用户: /skin routine weekly exfoliation mask

保存:
{
  "routine_type": "weekly",
  "steps": ["exfoliation", "mask"]
}
```

## 四、皮肤检查

### 示例 10: 自我检查
```
用户: /skin exam self 发现背部新增2个痣，外观正常

保存:
{
  "exam_type": "self",
  "findings": "发现背部新增2个痣，外观正常",
  "total_moles_count": 25,
  "suspicious_moles": 0
}
```

### 示例 11: 医生检查
```
用户: /skin exam dermatologist 2025-06-15 皮肤科检查，未发现异常

保存:
{
  "exam_type": "dermatologist",
  "date": "2025-06-15",
  "findings": "未发现异常"
}
```

## 五、日晒防护

### 示例 12: 防晒记录
```
用户: /skin sun protection daily 使用SPF30防晒霜

保存:
{
  "record_type": "protection",
  "protection_used": ["spf30"]
}
```

### 示例 13: 日晒伤
```
用户: /skin sun burn moderate 海边日晒2小时，中度晒伤

保存:
{
  "record_type": "burn",
  "burn_severity": "moderate",
  "exposure_duration_hours": 2,
  "exposure_level": "high"
}
```

## 六、查看状态

### 示例 14: 查看皮肤健康状态
```
用户: /skin status

输出:
 皮肤健康状态

皮肤类型：混合性
━━━━━━━━━━━━━━━━━━━━━━━━━━

当前问题：
━━━━━━━━━━━━━━━━━━━━━━━━━━
痤疮（中度）- 额头
色斑 - 脸颊

痣的监测：
━━━━━━━━━━━━━━━━━━━━━━━━━━
总数：25个
可疑：0个
下次自查：2025-07-10

护肤习惯：良好
━━━━━━━━━━━━━━━━━━━━━━━━━━
每天防晒：是 
每天护肤：是 

健康评分：75/100（良好）
```

## 七、趋势分析

### 示例 15: 查看趋势
```
用户: /skin trend 6months

输出:
 皮肤健康趋势（近6个月）

皮肤问题变化：
━━━━━━━━━━━━━━━━━━━━━━━━━━
痤疮：改善 
色斑：稳定 
痣的变化：无异常

护肤效果评估：
━━━━━━━━━━━━━━━━━━━━━━━━━━
保湿：改善 
控油：有效 
防晒：坚持良好 
```

## 八、检查提醒

### 示例 16: 查看提醒
```
用户: /skin reminder

输出:
 皮肤检查提醒

下次皮肤科检查：2025-12-15（还有180天）
━━━━━━━━━━━━━━━━━━━━━━━━━━

下次痣自查：2025-07-01（每月自查）
```
