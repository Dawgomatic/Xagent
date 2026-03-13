# travel-health 技能示例

## 一、旅行规划

### 示例 1: 东南亚旅行规划
```
用户: /travel plan Thailand Vietnam Cambodia 2025-08-01 for 14 days tourism

保存:
{
  "trip_id": "trip_20250801_seasia",
  "destination": ["Thailand", "Vietnam", "Cambodia"],
  "start_date": "2025-08-01",
  "end_date": "2025-08-15",
  "duration_days": 14,
  "travel_type": "tourism"
}

输出:
 旅行健康规划报告

目的地：泰国、越南、柬埔寨
旅行时间：2025-08-01 至 2025-08-15（14天）
旅行类型：旅游

风险评估： 中等风险
━━━━━━━━━━━━━━━━━━━━━━━━━━

推荐疫苗：
• 甲型肝炎疫苗
• 伤寒疫苗
• 日本脑炎疫苗（如农村地区）

推荐药物：
• 防疟疾药物（如前往边境地区）
• 防蚊虫叮咬措施

健康建议：
• 注意饮食安全
• 饮用瓶装水
• 避免生食
• 防蚊虫叮咬
```

### 示例 2: 商务旅行
```
用户: /travel plan Japan 2025-10-01 business

保存:
{
  "travel_type": "business",
  "destination": ["Japan"],
  "risk_assessment": {
    "overall_risk": "low"
  }
}
```

## 二、疫苗管理

### 示例 3: 记录疫苗接种
```
用户: /travel vaccine add hepatitis-a

保存:
{
  "vaccine_type": "hepatitis_a",
  "status": "scheduled",
  "trip_id": "trip_20250801_seasia"
}
```

### 示例 4: 更新接种状态
```
用户: /travel vaccine update hepatitis-a completed 2025-06-15

保存:
{
  "status": "completed",
  "date_administered": "2025-06-15"
}
```

### 示例 5: 查看接种计划
```
用户: /travel vaccine schedule

输出:
 疫苗接种计划

甲型肝炎疫苗：
━━━━━━━━━━━━━━━━━━━━━━━━━━
第一剂：2025-06-15 
第二剂：2025-12-15（6个月后）

伤寒疫苗：
━━━━━━━━━━━━━━━━━━━━━━━━━━
加强剂：2025-06-20 
下次：2026-06-20
```

## 三、旅行药箱

### 示例 6: 添加药箱物品
```
用户: /travel kit add antidiarrheal antibacterial

保存:
{
  "medications": [
    { "name": "antidiarrheal", "quantity": 2 },
    { "name": "antibacterial", "quantity": 1 }
  ]
}
```

### 示例 7: 检查药箱
```
用户: /travel kit check

输出:
 旅行药箱检查

药品：
━━━━━━━━━━━━━━━━━━━━━━━━━━
 止泻药 (2)
 退烧止痛药 (10)
 抗菌药 (需要补充)

医疗用品：
━━━━━━━━━━━━━━━━━━━━━━━━━━
 创可贴 (10)
 绷带 (2)
 体温计 (需要添加)

防护用品：
━━━━━━━━━━━━━━━━━━━━━━━━━━
 防晒霜 (1)
 驱蚊剂 (2)
```

## 四、用药管理

### 示例 8: 添加旅行用药
```
用户: /travel medication add doxycycline 100mg daily for malaria prophylaxis start 2025-07-28

保存:
{
  "name": "doxycycline",
  "dosage": "100mg",
  "frequency": "daily",
  "purpose": "malaria prophylaxis",
  "start_date": "2025-07-28"
}
```

### 示例 9: 检查药物相互作用
```
用户: /travel medication check-interactions

检查旅行用药之间是否有相互作用
```

## 五、准备状态

### 示例 10: 查看准备状态
```
用户: /travel status

输出:
 旅行准备状态

旅行：泰国、越南、柬埔寨（14天）
出发日期：2025-08-01

准备进度：75% 
━━━━━━━━━━━━━━━━━━━━━━━━━━

已完成：
 旅行规划
 疫苗接种（甲肝、伤寒）
 保险购买
 紧急联系人设置

待完成：
 防疟疾药物购买
 旅行药箱检查补充
 紧急卡片生成

距离出发还有：11天
```

## 六、风险评估

### 示例 11: 目的地风险评估
```
用户: /travel risk Thailand

输出:
 泰国旅行风险评估

综合风险： 中等风险
━━━━━━━━━━━━━━━━━━━━━━━━━━

主要风险：
━━━━━━━━━━━━━━━━━━━━━━━━━━
• 登革热：地方性流行
• 食物安全：中等风险
• 交通安全：中等风险
• 疟症：蚊虫传播疾病

疫苗建议：
━━━━━━━━━━━━━━━━━━━━━━━━━━
• 日本脑炎疫苗（农村地区）
• 甲型肝炎疫苗
• 伤寒疫苗

药物建议：
━━━━━━━━━━━━━━━━━━━━━━━━━━
• 防疟疾药物（如前往边境地区）

注意事项：
━━━━━━━━━━━━━━━━━━━━━━━━━━
• 防蚊虫叮咬（使用驱蚊剂）
• 饮用瓶装水
• 避免生食
• 注意食物卫生
```

## 七、保险信息

### 示例 12: 添加保险
```
用户: /travel insurance add policy123 $100000 covers medical_evacuation

保存:
{
  "policy_id": "policy123",
  "coverage_amount": 100000,
  "covers_medical_evacuation": true
}
```

## 八、紧急联系人

### 示例 13: 添加紧急联系人
```
用户: /travel emergency add spouse +86-138-xxxx-xxxx

保存:
{
  "name": "配偶",
  "relationship": "spouse",
  "phone": "+86-138-xxxx-xxxx"
}
```

## 九、紧急卡片

### 示例 14: 生成紧急卡片
```
用户: /travel card generate en zh th ja

输出:
 多语言紧急医疗信息卡片

已生成以下语言版本：
• 英文
• 中文
• 泰文
• 日文

内容包含：
• 基本信息
• 疾病状况
• 过敏信息
• 正在用药
• 血型
• 紧急联系人

文件：data/travel-health-cards/emergency_card_20250620.pdf
```

## 十、疫情预警

### 示例 15: 订阅预警
```
用户: /travel alert subscribe Thailand

保存:
{
  "subscribed_destinations": ["Thailand"],
  "last_checked": "2025-06-20"
}
```

### 示例 16: 查看预警
```
用户: /travel alert check

输出:
 目的地疫情预警

泰国（2025-06-20）：
━━━━━━━━━━━━━━━━━━━━━━━━━━
• 登革热：活跃
• 寨卡病毒：无报告
• 新冠肺炎：低风险

建议：继续旅程，注意防蚊
```
