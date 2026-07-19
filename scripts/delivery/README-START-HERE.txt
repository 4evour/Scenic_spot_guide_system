景区智能导览系统 - 可执行演示包
================================

一、如何启动

1. 将整个压缩包解压到磁盘，建议路径不要太深，例如 D:\ScenicGuide。
2. 双击 START-DEMO.bat。
3. 第一次启动需要联网安装 Python 依赖，并下载约 1.12GB 的语音识别模型。
4. 首次启动请预留约 3GB 磁盘空间和 10-30 分钟，具体取决于网络速度。
5. 启动完成后浏览器会打开：http://127.0.0.1:8080/digital-human#/login

二、是否必须配置大模型 Key

不必须。没有 Key 时，本地知识库、景点、路线、数据大屏、真实 Live2D 和基础文字问答仍可使用。
配置 Key 后，开放式回答、多轮归纳和语言自然度会更好。

需要配置时，右键 Configure-Online-LLM.ps1，选择“使用 PowerShell 运行”。
不要使用作者的 Key，也不要把自己的 Key 发给其他人。

三、联网功能

- 首次安装 Python 依赖：需要联网。
- 首次下载 SenseVoice 语音识别模型：需要联网，下载后可本地识别。
- Edge TTS 在线语音朗读：使用时需要联网，不需要大模型 Key。
- DeepSeek/Qwen 等外部大模型：需要联网并使用评委自己的 Key。

四、常见问题

- 启动失败：查看同目录 logs 文件夹中的日志。
- 端口冲突：重新双击 START-DEMO.bat，脚本会重启 8080 和 12393 端口服务。
- 数字人显示备用头像：确认压缩包完整，并检查 scenic-guide\static\digital-human\libs 和 live2d-models。
- 想完全清除在线大模型配置：运行 PowerShell 命令
  .\Configure-Online-LLM.ps1 -Clear

五、安全说明

交付包不包含作者的 API Key、数据库历史、聊天记录或本机配置。
评委自行配置的 Key 只保存在当前解压目录的 scenic-guide\.env.local。
