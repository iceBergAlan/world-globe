# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 构建与运行

```bash
# 本地开发
./start.sh        # 启动服务（需设置环境变量）
./restart.sh      # 重启服务

# 手动构建
go build -o server main.go
./server

# 部署（Railway 自动执行）
go build -o server main.go && ./server
```

## 环境变量

| 变量 | 必须 | 说明 |
|---|---|---|
| `MINIMAX_API_KEY` | 是 | MiniMax LLM API 密钥 |
| `PORT` | 否 | 监听端口，默认 3000 |
| `SITE_URL` | 否 | 站点地址，用于生成分享二维码 |
| `ZHIHU_COOKIE` | 否 | 知乎发布功能所需 Cookie |

## 架构

两个核心文件：

- **`main.go`**：Go 标准库 HTTP 服务器，无外部依赖
  - `POST /api/generate`：接收查询词 → 调用 MiniMax LLM → 解析国家列表 → 映射坐标 → 返回 `[]Item`
  - `GET /api/config`：返回 `SITE_URL` 供前端生成二维码
  - `POST /api/publish`：用 `ZHIHU_COOKIE` 调用知乎 API 发布问题
  - `countryMap`：内置约 40 个国家的经纬度坐标；同国家多标记点做螺旋偏移避免重叠

- **`static/index.html`**：纯 HTML 单文件前端，CDN 引入 `globe.gl`（3D 地球）、`html2canvas`（截图）、`qrcodejs`（二维码）

## 关键设计

- LLM 返回 JSON 数组，每项包含 `Country`（英文无空格）、`Name`、`Desc`、`Roast`、`Emoji`、`SearchQuery`
- 国家名通过 `countryMap` 映射为经纬度；未知国家会被过滤掉
- 前端拼接词面板：选择形容词 + 名词后自动触发搜索，名词列表根据形容词动态更新
- 部署平台：Railway（`railway.json` 配置构建和启动命令）
