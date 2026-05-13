# 公开仓库安全审查报告

## 执行摘要

本次审查面向仓库公开前的信息泄露风险。当前跟踪文件未发现高置信度真实密钥；本地真实配置、数据库、日志、构建调试映射和环境文件已被忽略；CI 与 Makefile 已加入密钥形态扫描；服务端补充了 HTTP 超时与基础安全响应头。

需要注意：Git 历史中曾出现过 API Key 形态文本，已在本地重写历史清理，但凡是进入过历史的真实密钥都应在服务商控制台吊销或轮换，不能只依赖历史清理。

## 关键结论

### [C-1] 历史曾包含 API Key 形态文本

影响：如果这些密钥曾经推送到任何远端或被他人克隆，攻击者可能仍可使用旧密钥调用相关服务。

状态：已处理历史文本清理，并重新执行历史密钥扫描，未发现 `sk-` 形态密钥残留。仍需人工在 DeepSeek、DashScope 等服务商控制台轮换或吊销对应密钥。

### [H-1] 本地真实配置文件必须保持未跟踪

位置：`.gitignore:37`、`.dockerignore:4`、`README.md:49`

状态：`configs/config.yaml`、`configs/digital_human.yaml` 已被 `.gitignore` 忽略，Docker build context 也排除了真实配置。当前本地 `configs/config.yaml` 含真实密钥形态内容，因此公开仓库前不要手动强制添加该文件。

### [H-2] Git 历史仍残留旧 sourcemap 文件

位置：`web-vue/vite.config.ts:10`、`.gitignore:50`、`.dockerignore:14`

状态：当前版本已关闭前端 sourcemap，当前跟踪文件不包含 `.map` 构建产物；历史中的旧 `.map` 文件仍需要通过历史重写清除后再推送公开仓库。

## 已完成加固

- `.github/workflows/ci.yml:48` 增加 `Secret scan`，CI 会运行高置信度密钥扫描。
- `Makefile:3` 将密钥扫描纳入 `make check`。
- `scripts/check-secrets.mjs:26` 检查常见 API Key、GitHub Token、Google API Key、AWS Access Key、私钥和 Bearer Token 形态。
- `main.go:85` 为 HTTP Server 设置请求头、读、写、空闲超时和最大请求头大小，降低慢请求耗尽风险。
- `internal/handler/routes.go:13` 增加统一安全响应头；`internal/handler/routes.go:59` 保留同源摄像头、麦克风和屏幕捕获能力，避免破坏数字人页面，同时禁用地理位置。
- `README.md:69` 记录公开或提交前运行密钥扫描的命令。

## 剩余风险

### [M-1] 前端 JWT 存储在 localStorage

位置：`static/js/app.js:423`、`web-vue/src/views/AdminView.vue:195`

说明：当前实现使用 `localStorage.authToken` 保存 JWT。公开仓库本身不会泄露密钥，但若页面出现 XSS，Token 更容易被读取。生产环境建议迁移到 `HttpOnly`、`Secure`、`SameSite` Cookie，并配套 CSRF 防护。

### [M-2] 静态数字人资源体积大且包含第三方运行库

位置：`static/digital-human/libs/live2dcubismcore.min.js`、`static/digital-human/libs/*.wasm`

说明：这些文件不是密钥泄露，但公开前应确认第三方资源、模型和运行库的许可证允许公开分发。

## 验证记录

- `node scripts/check-secrets.mjs`：通过。
- `go test ./...`：通过。
- `go vet ./...`：通过。
- `npm run check:encoding`：通过。
- `npm run check`：通过。
- `npm run build`：通过；Vite 仅提示非 module 外部 Live2D 脚本无法打包，属于现有静态引入方式。

## 公开前操作清单

1. 轮换或吊销所有曾写入本地配置或 Git 历史的 DeepSeek、DashScope 等 API Key。
2. 清理 Git 历史中的旧 sourcemap 文件。
3. 重跑当前文件与历史密钥扫描。
4. 使用 `git push --force-with-lease origin master` 推送重写后的历史。
5. 若远端已被他人克隆，通知协作者重新克隆或按新历史重置本地分支。
