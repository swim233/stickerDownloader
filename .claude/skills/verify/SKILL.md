---
name: verify
description: 验证 StickerDownloader 的 Telegram Bot 回调处理
---

# StickerDownloader 验证

## 常规检查

从仓库任意目录运行：

```bash
go -C src test ./...
```

## Telegram 回调端到端验证

Bot 通过长轮询接收更新。安全验证 handler 时，应使用本地 Telegram Bot API 模拟器和假 token：

1. 构建当前工作树的真实二进制，不修改源码，也不要复制真实 `config/config.yaml`。
2. 用最小假配置启动应用（HTTP server disabled），将 `telegram.api_endpoint` 指向本地模拟器。
3. 模拟 `getMe`、`getUpdates`、`answerCallbackQuery` 以及目标 handler 需要的 API 响应。
4. 通过 `Bot.Run` 的真实更新分发链路发送 callback update，并捕获应用日志与 API 请求。
5. 至少覆盖目标场景及一个相邻场景，例如缺失 `reply_to_message` 与结构完整的回调。

注意：模拟器若立即返回空 `getUpdates`，应用会高速轮询并产生大量日志；输出时只截取目标 endpoint 和 handler 日志。
