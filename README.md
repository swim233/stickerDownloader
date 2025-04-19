# 🧊 Sticker Downloader Bot

一个用于 **Telegram Bot** 下载贴纸与贴纸包的工具，支持多种格式转换与 HTTP API 接口。

## ✅ 功能支持

- ✔️ 在私聊中下载贴纸  
- ✔️ 支持下载 `.webm` 格式的贴纸  
- ✔️ 支持下载整个贴纸包并自动打包为 `.zip`  
- ✔️ 支持贴纸格式转换（支持 `webp` / `png` / `jpeg`）  
- ❌ ~~支持下载进度条~~  
- ✔️ 提供 HTTP API 服务  
- ✔️ 启用贴纸缓存，加快重复下载速度  
- ⏳ 支持用户自定义输出文件名格式（开发中）  
- ⏳ 多语言支持（开发中）

---

## 🚀 快速开始

第一次运行时会自动生成 `.env` 文件，请配置以下内容：

```env
# Telegram Bot Token
Token=YOUR_TOKEN_ID

# HTTP Server Telegram Bot Token
HTTPToken=YOUR_TOKEN_ID

# 日志等级 (可选值: DEBUG, INFO, WARN, ERROR)
LogLevel=DEBUG/INFO/WARN/ERROR

# 是否开启BotAPI debug输出(true/false)
DebugFlag=true

# API 日志等级 (可选值: DEBUG, INFO, WARN, ERROR)
ApiLogLevel=DEBUG/INFO/WARN/ERROR

# WebP 转 JPEG 的质量 (范围: 0-100)
WebPToJPEGQuality=100

# HTTP 服务器端口 (格式: :端口号)
HTTPServerPort=:8070

# 是否启用 HTTP 服务器 (true/false)
EnableHTTPServer=false

# 是否启用缓存 (true/false)
EnableCache=true

# 缓存过期时间 (单位: 分钟)
CacheExpirationTime=120
```

HTTP 服务器默认监听端口为：`8070`

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
- [ ] 多语言界面支持  

---

如果你觉得这个项目对你有帮助，欢迎 ⭐️ Star 或提 PR！
