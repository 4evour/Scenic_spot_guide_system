# 生产数据库方案

## 结论

真实景区商用部署必须使用 PostgreSQL。SQLite 只用于本地开发、离线演示或临时验证，不作为生产数据库。

项目当前已经支持：

- `database.driver: postgres`
- Docker Compose 中的 PostgreSQL 服务
- 通过环境变量注入数据库连接信息
- GORM 模型和 `AutoMigrate`

生产上线时，`AutoMigrate` 不能作为唯一迁移机制。后续结构变更需要显式迁移脚本和回滚策略。

## 环境划分

### 本地开发

- 可使用 SQLite。
- 默认路径：`./data/scenic_guide.db`。
- 数据可随开发环境重建，不承诺长期保留。

### 生产环境

- 必须使用 PostgreSQL。
- 数据库账号、密码、主机和库名通过环境变量或部署密钥注入。
- 数据库存储必须独立于应用容器生命周期。
- Docker 部署时不得把 PostgreSQL volume 当作备份。

## 持久化数据

以下数据进入 PostgreSQL 长期存储：

- 用户账号
- 游客账号
- 游客升级后的账号关系
- 数字人偏好
- 聊天会话
- 聊天消息
- 行为日志
- 游客反馈
- 管理端配置
- 知识库数据

游客升级为正式用户时，应保留同一个 `user_id`，避免聊天记录和偏好断链。

## 备份与恢复

生产部署必须配置数据库备份：

- 每日至少一次全量备份。
- 关键部署建议开启 WAL 归档或云数据库 PITR。
- 备份文件应保存在独立存储位置。
- 每次正式上线前至少执行一次恢复演练。
- 备份恢复步骤需要由景区运维或交付团队可执行。

最低恢复演练清单：

1. 新建空 PostgreSQL 实例。
2. 从最近备份恢复数据。
3. 启动应用服务。
4. 验证管理员登录、游客登录、游客升级和聊天历史查询。

## 数据保留策略

建议第一版采用保守默认值：

| 数据 | 建议保留时间 |
| --- | --- |
| 用户账号 | 长期保留，直到用户删除或按合同/合规要求处理 |
| 数字人偏好 | 随账号长期保留 |
| 聊天会话 | 180 天或 365 天 |
| 聊天消息 | 180 天或 365 天 |
| 行为日志 | 90 天或 180 天 |
| 管理端聚合统计 | 长于原始日志 |
| 知识库数据 | 长期保留 |

原始日志增长最快，后续应增加定期归档或清理任务。管理端统计应优先使用聚合数据，避免长期直接扫描原始日志表。

## 索引检查

上线前至少检查以下查询路径：

- `users.username`
- `users.guest_token`
- `chat_sessions.user_id`
- `chat_sessions.last_active_at`
- `chat_messages.user_id`
- `chat_messages.created_at`
- `interaction_logs.user_id`
- `interaction_logs.source`
- `interaction_logs.created_at`

如果后台统计接口出现慢查询，优先增加组合索引或日聚合表，不直接扩大应用实例数量。

## 连接池

当前 Go 服务支持数据库连接池配置：

- `SCENIC_GUIDE_DATABASE_MAX_OPEN_CONNS`
- `SCENIC_GUIDE_DATABASE_MAX_IDLE_CONNS`
- `SCENIC_GUIDE_DATABASE_CONN_MAX_LIFETIME_MINUTES`

生产初始值可以沿用 Docker Compose 中的配置，再根据数据库监控调整。不要把连接数设置得远高于 PostgreSQL 实例承载能力。

## 上线前检查

- [ ] 生产环境 `SCENIC_GUIDE_DATABASE_DRIVER=postgres`
- [ ] PostgreSQL 数据目录或云数据库存储已持久化
- [ ] 数据库密码通过密钥注入，不提交到代码仓库
- [ ] 备份任务已配置
- [ ] 恢复演练已完成
- [ ] 账号、游客升级、修改密码、聊天历史查询流程已验证
- [ ] 慢查询和数据库容量有监控入口
