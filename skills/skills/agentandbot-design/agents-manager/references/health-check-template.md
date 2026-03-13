# Health Check Template

Health check queries and status codes for agent monitoring.

## Standard Health Query

Send this query to each agent to get a standardized response:

```
"Health check: Respond with:
- Status: healthy|slow|error|offline
- Model: <your model>
- Last active: <timestamp>
- Tasks completed today: <number>"
```

## Status Codes

| Status | Meaning | Emoji | Action |
|---------|-----------|--------|---------|
| **healthy** | Agent responding normally |  | No action needed |
| **slow** | Responding but delayed >5s |  | Monitor closely |
| **error** | Returning errors |  | Investigate and restart |
| **offline** | Not responding |  | Check agent status |

## Response Time Thresholds

| Category | Threshold |
|-----------|------------|
|  Healthy | < 5 seconds |
|  Slow | 5-30 seconds |
|  Error | > 30 seconds |
