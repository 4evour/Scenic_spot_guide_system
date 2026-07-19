# Live2D 本地打包与启动设计

## 目标

本地评委演示必须同时显示真实 Live2D 模型。`scripts/start-local.ps1 -Restart` 不得在 Cubism Core 或模型资源缺失时把备用动效视为启动成功。

## 方案

采用启动时部署方案。源码包必须同时包含同级的 `scenic-guide/` 与 `Open-LLM-VTuber/`：

1. Cubism Core 来源为 `Open-LLM-VTuber/frontend/libs/live2dcubismcore.min.js`，部署到 `scenic-guide/static/digital-human/libs/live2dcubismcore.min.js`。
2. Live2D 模型来源为 `Open-LLM-VTuber/live2d-models/`，部署到 `scenic-guide/static/live2d-models/`。
3. 启动脚本在初始化数据库和启动 Go 服务之前完成部署，并校验 Core、模型配置和 `.moc3` 文件。
4. 任一核心资源缺失时立即终止启动，输出缺失路径；不继续显示备用头像冒充完整数字人。
5. 资源只保留一个源目录。主系统静态目录中的副本由启动脚本生成，不作为另一套人工维护的模型源。

## 边界

- 不修改 `Open-LLM-VTuber` 的 Python、配置或模型内容。
- 不修改现有 Vue Live2D 加载协议和模型 URL。
- 打包方负责确认 Cubism Core 与模型具备演示和分发授权。
- 仅 `start-local.ps1` 的本地演示流程强制部署；生产部署继续按正式发布流程提供静态资产。

## 验证

- 脚本静态检查确认资源部署函数、三个必需文件检查和启动前调用存在。
- 使用临时目录验证缺失资源会失败、完整资源会复制成功。
- 执行前端类型检查、lint 和生产构建，以及 Go 全量测试、vet 和构建。
- 运行 `scripts/start-local.ps1 -Restart -NoBrowser`，确认 Core、模型 URL 返回 200。
- 浏览器确认 `window.Live2DCubismCore` 存在、页面不显示 SDK 降级提示，并显示真实 Live2D 模型。

