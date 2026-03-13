# oral-health 技能示例

## 一、检查记录

### 示例 1: 基础检查记录
```
用户: /oral checkup 2025-06-10 牙齿检查，16号牙有龋齿，牙周健康

保存:
{
  "id": "checkup_20250610_001",
  "date": "2025-06-10",
  "teeth_status": {
    "16": { "condition": "caries", "note": "龋齿" }
  },
  "periodontal_status": {
    "classification": "healthy"
  }
}
```

### 示例 2: 详细检查记录
```
用户: /oral checkup 今日口腔检查，牙龈无出血，牙齿状态良好

保存:
{
  "date": "2025-06-10",
  "periodontal_status": {
    "bleeding_on_probing": false,
    "classification": "healthy"
  },
  "soft_tissue": "normal"
}
```

## 二、治疗记录

### 示例 3: 补牙治疗
```
用户: /oral treatment filling 26号牙 复合树脂充填，费用300元

保存:
{
  "id": "treatment_20250610_001",
  "date": "2025-06-10",
  "tooth_number": "26",
  "treatment_type": "filling",
  "material": "composite_resin",
  "cost": 300
}
```

### 示例 4: 根管治疗
```
用户: /oral treatment root_canal 46号牙 根管治疗，2次就诊

保存:
{
  "tooth_number": "46",
  "treatment_type": "root_canal",
  "description": "根管治疗，2次就诊"
}
```

### 示例 5: 拔牙
```
用户: /oral extraction 18号牙 智齿拔除，术后恢复良好

保存:
{
  "tooth_number": "18",
  "treatment_type": "extraction",
  "description": "智齿拔除，术后恢复良好"
}
```

## 三、卫生习惯

### 示例 6: 记录刷牙习惯
```
用户: /oral hygiene brushing twice_daily 每天2次，每次2分钟

保存:
{
  "brushing": {
    "frequency": "twice_daily",
    "duration_minutes": 2
  }
}
```

### 示例 7: 记录使用牙线
```
用户: /oral hygiene flossing daily 每天使用牙线

保存:
{
  "flossing": {
    "frequency": "daily"
  }
}
```

### 示例 8: 记录漱口水使用
```
用户: /oral hygiene mouthwash sometimes 偶尔使用漱口水

保存:
{
  "mouthwash": {
    "frequency": "few_times_week"
  }
}
```

## 四、口腔问题

### 示例 9: 牙痛记录
```
用户: /oral issue toothache 46号牙，冷热敏感，中度疼痛

保存:
{
  "issue_type": "toothache",
  "tooth_number": "46",
  "severity": "moderate",
  "description": "冷热敏感"
}
```

### 示例 10: 牙龈出血
```
用户: /oral issue bleeding 牙龈刷牙时出血，持续3天

保存:
{
  "issue_type": "bleeding",
  "severity": "mild",
  "duration": "3天",
  "description": "刷牙时出血"
}
```

### 示例 11: 口腔溃疡
```
用户: /oral issue ulcer 下唇溃疡，疼痛影响进食

保存:
{
  "issue_type": "ulcer",
  "severity": "moderate",
  "description": "下唇溃疡，疼痛影响进食"
}
```

## 五、查看状态

### 示例 12: 查看口腔健康状态
```
用户: /oral status

输出:
 口腔健康状态

最近检查：2025-06-10
下次检查：2025-12-10

牙齿状态：
━━━━━━━━━━━━━━━━━━━━━━━━━━
16号牙：龋齿 
其他：正常

卫生习惯评分：8/10（良好）
━━━━━━━━━━━━━━━━━━━━━━━━━━
刷牙：每天2次 
使用牙线：每天 

活跃问题：无
```

## 六、趋势分析

### 示例 13: 查看趋势
```
用户: /oral trend 6months

输出:
 口腔健康趋势（近6个月）

龋齿发展趋势：
━━━━━━━━━━━━━━━━━━━━━━━━━━
新增龋齿：1颗（16号牙）
需要治疗：1颗

牙周健康变化：
━━━━━━━━━━━━━━━━━━━━━━━━━━
牙龈出血：减少 
探诊深度：稳定

卫生习惯改善：
━━━━━━━━━━━━━━━━━━━━━━━━━━
刷牙频率：保持良好
使用牙线：从偶尔→每天 
```

## 七、检查提醒

### 示例 14: 查看提醒
```
用户: /oral reminder

输出:
 口腔检查提醒

下次检查：2025-12-10（还有180天）
━━━━━━━━━━━━━━━━━━━━━━━━━━

检查项目：
• 牙齿龋齿检查
• 牙周状况评估
• 软组织检查
• 咬合评估

准备事项：
• 携带既往X光片
• 列出用药清单
• 准备问题描述
```

### 示例 15: 设置提醒
```
用户: /oral reminder set 2025-12-10

更新下次检查提醒日期
```

## 八、疾病筛查

### 示例 16: 龋齿风险评估
```
用户: /oral screening caries 龋齿风险评估

输出:
龋齿风险评估
━━━━━━━━━━━━━━━━━━━━━━━━━━
当前风险：中风险

风险因素：
 糖摄入：中等
 卫生习惯：一般
 使用含氟产品：是
 定期检查：是

建议：
• 减少糖摄入
• 改善刷牙习惯
• 考虑使用含氟漱口水
```
