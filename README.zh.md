# Cachembed

一个轻量级的缓存代理，用于 OpenAI 嵌入 API 请求。

## 概述

Cachembed 是一个代理服务器，缓存 OpenAI 嵌入 API 的结果，以减少冗余请求并降低成本。它支持 SQLite（默认）和 PostgreSQL 作为存储后端。

## 特性

- 将嵌入结果缓存到 SQLite 或 PostgreSQL
- 代理请求到 OpenAI API（默认网址为 https://api.openai.com/v1/embeddings）
- 支持通过正则表达式模式验证 API 密钥
- 限制使用已允许的嵌入模型
- 支持数据库迁移
- 可通过环境变量进行配置

## 要求

* Ruby 3.4.1 或更高版本
* Rails 8.0.1 或更高版本
* SQLite3 或 PostgreSQL

## 安装

克隆代码库并安装依赖项：

```bash
git clone https://github.com/your-username/cachembed-rails
cd cachembed-rails
# 如果您想使用 PostgreSQL，请运行：
bundle install --with=postgresql
# 如果您想使用 SQLite，请运行：
bundle install
```

## 设置

创建并迁移数据库：

```bash
bin/setup --skip=server
```

## 配置

使用以下环境变量配置应用程序：

| 环境变量                       | 描述                                     | 默认值                                          |
|------------------------------|----------------------------------------|------------------------------------------------|
| CACHEMBED_UPSTREAM_URL       | OpenAI 嵌入 API 端点                      | https://api.openai.com/v1/embeddings          |
| CACHEMBED_ALLOWED_MODELS     | 允许模型的逗号分隔列表                     | text-embedding-3-small,text-embedding-3-large,text-embedding-ada-002 |
| CACHEMBED_API_KEY_PATTERN    | API 密钥验证的正则表达式模式              | ^sk-[a-zA-Z0-9_-]+$                           |
| DATABASE_URL                 | 数据库连接字符串                          | 取决于 config/database.yml                     |

## 使用

### 启动服务器

开发环境：

```bash
rails server
```

生产环境：

```bash
RAILS_ENV=production rails server
```

### API 端点

服务器提供以下端点：

- POST `/v1/embeddings`：代理请求到 OpenAI 的嵌入 API，并进行缓存

示例请求：

```bash
curl -X POST http://localhost:3000/v1/embeddings \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-api-key" \
  -d '{
    "input": "Your text here",
    "model": "text-embedding-3-small"
  }'
```

## 许可证

MIT 许可证

## 贡献

欢迎提交拉取请求！如果您发现错误或想请求某个功能，请提出问题。

## TODO

- 实现 LRU 缓存（带请求日志）和旧缓存条目的垃圾回收