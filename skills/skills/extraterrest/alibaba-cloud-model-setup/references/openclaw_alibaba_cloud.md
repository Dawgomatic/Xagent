# OpenClaw + Alibaba Cloud Model Studio Reference

## Endpoint mapping

- `Beijing / China site / CN`: `https://dashscope.aliyuncs.com/compatible-mode/v1`
- `Singapore / International site / INTL`: `https://dashscope-intl.aliyuncs.com/compatible-mode/v1`
- `Virginia / US site / US`: `https://dashscope-us.aliyuncs.com/compatible-mode/v1`

## Live models API

- Model list endpoint: `GET <base-url>/models`
- Requires: `Authorization: Bearer <DASHSCOPE_API_KEY>`
- Returns OpenAI-compatible payload where model IDs are in `data[].id`

Example:

```bash
curl -sS https://dashscope-intl.aliyuncs.com/compatible-mode/v1/models \
  -H "Authorization: Bearer $DASHSCOPE_API_KEY" \
  -H "Content-Type: application/json"
```

## Site guidance links

- International site console (Singapore): `https://modelstudio.console.alibabacloud.com/ap-southeast-1`
- International site console (US Virginia): `https://modelstudio.console.alibabacloud.com/us-east-1`
- China site overview: `https://help.aliyun.com/zh/model-studio/what-is-model-studio?`

## Provider defaults used by this skill

- Provider name: `balian`
- API mode: `openai-completions`
- Default preset models:
- `qwen-max`
- `qwen-flash`
- `qwen3-coder-plus`
- Primary model default: `qwen3-coder-plus`
- Reasoning: `false`

## Config structure touched

- `models.providers.<provider>`
- `agents.defaults.model.primary` (when default model is enabled)
- `agents.defaults.models.<provider/model>` (when default model is enabled)

## Notes

- Base URL, API key, and model should belong to the same region.
- For safer storage, set provider `apiKey` to `${DASHSCOPE_API_KEY}` (or another env var) and export it in the service runtime environment.
- This skill can persist env vars to `~/.bashrc` and to a systemd user service override.
- If OpenClaw command is unavailable, user may still be on historical binary names (`moltbot` or `clawdbot`).
- If JSON parsing fails, preserve file and switch to manual fix flow instead of blind overwrite.
