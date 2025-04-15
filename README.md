# stickerDownloader
在telegram bot中下载贴纸和贴纸包

- [x] 在私聊中下载贴纸
- [x] 支持下载webm格式贴纸
- [x] 支持下载贴纸包并打包发送
- [x] 支持格式转换
- [ ] ~~支持下载进度条~~
- [x] 支持HTTPServer
- [ ] 贴纸缓存
- [ ] 支持用户配置输出文件名格式
- [ ] 支持修改语言

- 第一次运行会自动生成`.env`文件 请自行配置`BotToken` `http服务器端口`和日志等级 
- http服务默认端口为 ``8070``

API 端点
1. 获取贴纸包的 JSON 信息
- URL: /stickerpack
- 方法: GET
- 参数:
  - name (必需): 贴纸包的名称。
  - download (可选): 如果为 true，则返回 ZIP 文件；否则返回 JSON 信息。
- 示例:
```curl "http://localhost:8070/stickerpack?name=sticker_pack_name"```

- 返回:
  - 成功时返回贴纸包的 JSON 信息。
  - 如果 name 参数缺失，返回 400 Bad Request。

2. 下载贴纸包为 ZIP 文件
- URL: /stickerpack
- 方法: GET
- 参数:
  - name (必需): 贴纸包的名称。
  - download (必需): 设置为 true 以下载 ZIP 文件。
  - format(可选):从webp/png/jpeg选择一个来决定下载的格式
- 示例:
```curl -o stickerpack.zip "http://localhost:8070/stickerpack?name=sticker_pack_name&download=true&format=webp"```

- 返回:
  - 成功时返回 ZIP 文件。
  - 如果 name 参数缺失，返回 400 Bad Request。
  - 如果 format 参数缺失，默认使用webp格式。
