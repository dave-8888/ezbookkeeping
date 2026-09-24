# 服务端 AI 对话（Android 0.6.0）

## 架构与职责

手机 → 已登录的对话 API → 服务内 AI 编排 → 受当前用户权限约束的业务接口 → AI 整理结果 → 手机展示。

模型调用、工具定义、工具执行循环、原生模型消息、资产序号上下文均留在后端。安卓不解释 tool calls，不传模型消息，不为每次对话读取 AI 配置，也不根据关键词决定业务动作。App 设置中的 AI 配置是服务端配置的管理入口，推理仍由服务执行。

当前执行通道直接复用服务内部现有业务处理接口，支持 `list_accounts`、`query_transactions`、`get_transaction_options`、`set_account_balance`、`create_transaction`。尚未新增通用远程 MCP 连接器。

## 手机接口

使用现有 Bearer 登录认证及客户端时区头。

`POST /api/v1/llm/conversations/respond.json`

```json
{"conversationId":"conversation-uuid","messageId":"message-uuid","text":"输出我的所有资产"}
```

统一 API 包装中的 result：

```json
{"messageId":"message-uuid","status":"completed","reply":"当前资产 · 不含隐藏账户","details":"1. 钱包\n37.42 CNY"}
```

可选 `verified` 是实际写入并回读后的确认信息。公开响应不包含模型协议、密钥或工具调用记录。查询成功而后续模型生成失败时，仍返回已取得的真实资产或账单结果。无有效结果时显示简短中文提示，供应商诊断记录在服务日志。

`GET /api/v1/llm/conversations/result.json?conversation_id=...&message_id=...`

用于断线后“查看处理结果”，只读取持久化结果，不重放操作。状态包括 completed、processing、interrupted、unknown、busy、not_found。同一用户相同消息 ID 的重复 POST 返回已存结果；同一 ID 更换文本会拒绝。同一对话同时只处理一条消息。

## 数据与执行约束

新增服务端 ConversationSession、ConversationTurn 表，按登录用户隔离上下文、执行状态与结果。上下文最多选取最近 12 轮，另受消息数与大小限制。资产序号引用服务端最近一次列表，切换 conversationId 不继承序号。

工具复用现有账户、分类、交易权限及校验。金额使用定点整数；各币种分别汇总。余额调整检查列表快照，使用 expectedBalance 原子校验，随后回读确认，不生成虚假收入或支出。每条消息最多尝试一次写入；结果不确定时停止执行并保留待核实状态。进程中断的过期任务不会自动重放。

聊天展示仍缓存在手机。删除对话、清空本机聊天仅删除手机记录，不删除服务端上下文或账目。退出登录保留聊天缓存并清除令牌。当前没有服务端对话删除或自动过期清理接口。

## 升级与验证

先备份数据库，构建并更新后端，执行项目原有数据库升级命令 `financially_free database update`，再安装 0.6.0 APK。新增表通过 cmd/database.go 的正常迁移入口创建。反向代理需允许长请求：服务最多进行 8 轮模型调用，单次模型请求最多 60 秒，App 对话请求超时为 9 分钟。

0.6.0 需要新 respond/result 接口，旧后端不支持，不能只升级 APK。旧 chat.json 暂保留兼容旧客户端。旧本机模型上下文不上传；升级后需要重新查询资产列表才能使用序号修改。

回归覆盖：服务端重复消息、跨用户隔离、过期任务恢复、模型汇总失败保留查询结果、多步执行与未知写入停止；真实账户接口的权限、币种、序号及余额并发校验；安卓只提交公开消息结构、断线 GET 查询与对话 ID 延续。
