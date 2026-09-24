# 网页端 MCP 直接记账

网页悬浮「添加交易」文本入口（桌面和移动网页版）调用 `POST /api/v1/llm/transactions/create_by_mcp.json`，提交 `text` 和 `requestId`。后端使用当前用户的 AI 配置进行原生工具调用，并通过项目 MCP 注册表执行 `query_all_accounts`、`query_all_transaction_categories`、`query_all_transaction_tags` 和 `add_transaction`。

这是后端进程内桥接，与公开 `/mcp` 使用同一套工具定义和处理器，不需要浏览器保存 MCP token，也不要求开启对外 MCP HTTP 服务。用户的 AI 使用限制和 MCP 访问限制均生效。所选模型需要支持工具调用。

成功后直接入账并刷新交易、账户、概览和统计缓存，不再由浏览器将识别结果另行保存。每次请求最多创建一笔；必要信息不足时保留输入供修改后重新提交。普通手动交易表单、图片识别及其他识别入口不受此修改影响。

日期上下文采用客户端时区，包含当前完整时间、星期、今天、昨天和前天。MCP 写入要求带时区偏移的 RFC3339 时间，并保存该交易的时区偏移。没有具体出错样例时，无法据此保证所有自然语言日期都被模型正确理解。

请求身份保存在浏览器会话存储，服务端复用现有 conversation 表持久化去重记录。超时、刷新后再次提交相同文本会复用未完成的请求身份；服务端返回原结果，不重放写入。写入开始前保存检查点；结果不确定时提示核对交易列表，不自动重试。关闭浏览器会话或修改文本会形成新请求，不属于原请求的重试。

验证：`go test ./pkg/api ./pkg/mcp ./pkg/services ./cmd`；前端 `quick_transaction_text_input` 和 `transaction_text` 测试；`vue-tsc --noEmit`。自动测试不调用真实模型，也不写入用户账本。
