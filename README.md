# 🧊 Sticker Downloader Bot

一个用于 **Telegram Bot** 下载贴纸与贴纸包的工具，支持多种格式转换与 HTTP API 接口。

## ✅ 功能支持

- ✔️ 在私聊中下载贴纸  
- ✔️ 支持下载 `.webm` 格式的贴纸  
- ✔️ 支持下载整个贴纸包并自动打包为 `.zip`  
- ✔️ 支持贴纸格式转换（支持 `webp` / `png` / `jpeg`）  
- ⏳ 支持下载进度条
- ✔️ 提供 HTTP API 服务  
- ✔️ 启用贴纸缓存，加快重复下载速度  
- ✔️ 支持数据库记录用户数据
- ✔️ 支持限制并发数
- ⏳ 支持用户自定义输出文件名格式（开发中）  
- ✔️ 多语言支持

---

## 🚀 快速开始

复制安全的示例配置并填写 Bot Token 与 Owner Chat ID：

```bash
cp config/config.example.yaml config/config.yaml
chmod 600 config/config.yaml
cd src
go build -o ../bin/stickerDownloader .
../bin/stickerDownloader --config ../config/config.yaml
```

程序默认以 supervisor 模式启动同一二进制的 worker 子进程。worker 意外退出时会指数退避并自动重启；5 分钟内连续崩溃 5 次会暂停 15 分钟，防止 crash loop。

在 `config/config.yaml` 中配置：

```yaml
telegram:
  token: "YOUR_BOT_TOKEN"
  http_token: "YOUR_BOT_TOKEN"
  owner_chat_id: 123456789
```

- `owner_chat_id` 仅从 YAML 读取；设为 `0` 会禁用运维通知，但不影响自动重启。
- Owner 会收到启动、任务级 panic、worker 崩溃、计划重启、崩溃循环和正常停止通知。
- 向 supervisor 发送 `SIGINT` 或 `SIGTERM` 会转发给 worker 并正常停止，不会触发重启。
- 推荐始终使用绝对路径传入 `--config`，避免工作目录变化导致读取错误。

完整的通知、重启和 HTTP 配置见 `config/config.example.yaml`。HTTP 服务器默认监听端口为 `8070`。

---

## 📦 API 端点文档

### 1. 获取贴纸包的 JSON 信息

- **URL**: `/stickerpack`
- **方法**: `GET`
- **参数**:
  - `name` (必需): 贴纸包名称（英文名）。
  - `download` (可选): 若为 `true`，将下载 `.zip` 文件，否则返回 JSON。
- **示例**:
```bash
curl "http://localhost:8070/stickerpack?name=sticker_pack_name"
```
- **返回**:
  - 成功时：贴纸包的 JSON 数据。
  - 缺失 `name` 参数：返回 `400 Bad Request`。

---

### 2. 下载贴纸包为 ZIP 文件

- **URL**: `/stickerpack`
- **方法**: `GET`
- **参数**:
  - `name` (必需): 贴纸包名称。
  - `download` (必需): 设置为 `true` 以下载压缩包。
  - `format` (可选): 指定输出格式，支持 `webp` / `png` / `jpeg`，默认为 `webp`。
- **示例**:
```bash
curl -o stickerpack.zip "http://localhost:8070/stickerpack?name=sticker_pack_name&download=true&format=webp"
```

---

## 🔗 演示链接

- 🤖 Telegram Bot: [@TheSw1mStickerDownloader_bot](https://t.me/TheSw1mStickerDownloader_bot)  
- 🌐 HTTP API: [http://oracle.swimgit.top:8070/](http://oracle.swimgit.top:8070/)

---

## 🛠️ TODO

- [ ] 支持用户自定义输出文件名格式
- [ ] 支持下载进度条  


---

如果你觉得这个项目对你有帮助，欢迎 ⭐️ Star 或提 PR！
