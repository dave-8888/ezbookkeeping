# AI 厂商与模型配置

在「应用设置 → AI」启用 AI，选择厂商，填写对应 API 密钥，然后点击「刷新模型」。模型列表支持搜索选择，也可以直接填写模型 ID，最后保存。

内置国内厂商：DeepSeek、阿里云百炼、火山引擎、腾讯混元、百度千帆、智谱、月之暗面、MiniMax。原有 OpenAI、OpenAI 兼容、OpenRouter、Ollama、Google AI 继续支持。国内厂商使用 OpenAI 兼容协议；既有兼容接口配置会按地址识别厂商，无需迁移数据库。

## 动态维护厂商目录

默认目录在 `pkg/llm/catalog/providers.json`，编译进后端。部署后需要动态更新时：

1. 将该文件复制到服务器可读取的路径，例如 `/etc/financially_free/ai-providers.json`。
2. 在实际使用的 INI 配置中设置：

   ```ini
   [llm]
   provider_catalog_file = /etc/financially_free/ai-providers.json
   ```

3. 首次启用这个配置项后重启后端。此后修改 JSON 文件无需重启服务、重新编译或发布前端。
4. 页面点击「刷新厂商」，或重新打开 AI 设置页，即可获取更新。

外部目录替换整个内置目录，可增加、修改、删除厂商；建议保留通用 `openai_compatible` 项，便于填写自定义地址。目录最多 100 个条目、1 MiB，ID 必须唯一。建议用临时文件写入完成后原子替换，避免读到半写入的 JSON。

```json
[
  {
    "id": "my-provider",
    "name": "自定义厂商",
    "protocol": "openai_compatible",
    "baseUrl": "https://example.com/v1",
    "docsUrl": "https://example.com/docs"
  },
  {
    "id": "openai_compatible",
    "name": "OpenAI Compatible",
    "protocol": "openai_compatible",
    "baseUrl": "",
    "docsUrl": ""
  }
]
```

`protocol` 支持 `openai`、`openai_compatible`、`openrouter`、`ollama`、`google_ai`。新增国内厂商通常选 `openai_compatible`，`baseUrl` 不包含 `/chat/completions`。预设 `baseUrl` 用于兼容接口，其他协议的连接地址沿用对应设置字段。目录仅放公开地址和名称，不存放 API 密钥。

刷新目录不会自动替换用户已经保存的地址、密钥或模型。若目录中的厂商被删除，原有连接仍保留为自定义配置。接口地址改变后，需要重新填写密钥；保存的密钥只会用于原地址。

## 模型刷新

后端使用当前表单的连接信息读取模型列表，不要求先保存，也不要求先填写模型 ID。密钥留空时，仅同一接口地址可以使用当前用户已保存的密钥。

- OpenAI、OpenRouter、OpenAI 兼容及国内预设：`GET <Base URL>/models`。
- Ollama：`GET <Server URL>/api/tags`。
- Google AI：`GET /v1beta/models`，读取分页并过滤支持 `generateContent` 的模型。

各厂商对模型目录接口的支持、返回范围和权限不一致。遇到不支持、鉴权失败、空列表或网络问题时，页面显示提示，并保留手动填写的模型 ID。刷新列表不会自动选择或覆盖模型，也不会发起推理请求。若返回列表没有能力标记，用户仍需选择支持文字对话的模型；图片识别还要求模型支持图像输入。

厂商名称是连接预设，不代表该厂商所有模型都兼容所有功能。火山方舟可按控制台要求手填模型 ID 或推理接入点 ID。百炼预设使用仍可用的北京公共域名，业务空间或其他地域用户可按控制台替换地址，并使用相应地域密钥。腾讯混元既有接口仍保留，新服务迁移情况请参阅官方说明。

## 官方接口参考

- [DeepSeek API](https://api-docs.deepseek.com/)、[模型列表](https://api-docs.deepseek.com/api/list-models)
- [阿里云百炼 OpenAI 兼容接口](https://help.aliyun.com/zh/model-studio/compatibility-of-openai-with-dashscope)
- [火山方舟](https://www.volcengine.com/docs/82379/1795150)
- [腾讯混元 OpenAI 兼容接口](https://cloud.tencent.com/document/product/1729/111007)
- [百度千帆模型列表](https://cloud.baidu.com/doc/qianfan-api/s/Dmba8k71y)
- [智谱 OpenAI 兼容接口](https://docs.bigmodel.cn/cn/guide/develop/openai/introduction)
- [Moonshot 开放平台](https://platform.moonshot.cn/docs/)
- [MiniMax 文本生成](https://platform.minimaxi.com/docs/guides/text-generation)

## 验证范围

单元测试覆盖厂商目录默认项、文件热更新、配置错误、模型返回解析、请求失败、重定向阻止、密钥隔离、表单预设、手填模型保留和过期响应丢弃。未使用真实厂商密钥进行逐家在线推理验证。
