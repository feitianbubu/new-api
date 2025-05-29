<p align="right">
   <strong>中文</strong> | <a href="./README.en.md">English</a>
</p>
<div align="center">

![clinx](/web/public/logo.png)

# Clinx

🍥新一代大模型网关与AI资产管理系统

<a href="https://trendshift.io/repositories/8227" target="_blank"><img src="https://trendshift.io/api/badge/repositories/8227" alt="Calcium-Ion%2Fnew-api | Trendshift" style="width: 250px; height: 55px;" width="250" height="55"/></a>

<p align="center">
  <a href="https://raw.githubusercontent.com/Calcium-Ion/new-api/main/LICENSE">
    <img src="https://img.shields.io/github/license/Calcium-Ion/new-api?color=brightgreen" alt="license">
  </a>
  <a href="https://github.com/Calcium-Ion/new-api/releases/latest">
    <img src="https://img.shields.io/github/v/release/Calcium-Ion/new-api?color=brightgreen&include_prereleases" alt="release">
  </a>
  <a href="https://github.com/users/Calcium-Ion/packages/container/package/new-api">
    <img src="https://img.shields.io/badge/docker-ghcr.io-blue" alt="docker">
  </a>
  <a href="https://hub.docker.com/r/CalciumIon/new-api">
    <img src="https://img.shields.io/badge/docker-dockerHub-blue" alt="docker">
  </a>
  <a href="https://goreportcard.com/report/github.com/Calcium-Ion/new-api">
    <img src="https://goreportcard.com/badge/github.com/Calcium-Ion/new-api" alt="GoReportCard">
  </a>
</p>
</div>

## 📝 项目说明

> [!NOTE]  
> 本项目在[New API](https://github.com/QuantumNous/new-api)的基础上进行二次开发

> [!IMPORTANT]  
> - 当前版本为云一内部专用
> - 根据[《生成式人工智能服务管理暂行办法》](http://www.cac.gov.cn/2023-07/13/c_1690898327029107.htm)的要求，请勿对中国地区公众提供一切未经备案的生成式人工智能服务。

## 📚 文档

详细文档请访问：[https://dev.clinx.work/swag.html](https://dev.clinx.work/swag.html)

## 🏗 架构
![clinx-arch](/web/public/arch.png)



## ✨ 主要特性

New API提供了丰富的功能，详细特性请参考[特性说明](https://docs.newapi.pro/wiki/features-introduction)：

1. 🎨 全新的UI界面
2. 🌍 多语言支持
3. 💰 支持在线充值功能（ND支付）
4. 🔍 支持用key查询使用额度（配合[neko-api-key-tool](https://github.com/Calcium-Ion/neko-api-key-tool)）
5. 🔄 兼容原版One API的数据库
6. 💵 支持模型按次数收费
7. ⚖️ 支持渠道加权随机
8. 📈 数据看板（控制台）
9. 🔒 令牌分组、模型限制
10. 🤖 支持更多授权登陆方式（LinuxDO,Telegram、OIDC）
11. 🔄 支持Rerank模型（Cohere和Jina），[接口文档](https://docs.newapi.pro/api/jinaai-rerank)
12. ⚡ 支持OpenAI Realtime API（包括Azure渠道），[接口文档](https://docs.newapi.pro/api/openai-realtime)
13. ⚡ 支持Claude Messages 格式，[接口文档](https://docs.newapi.pro/api/anthropic-chat)
14. 支持使用路由/chat2link进入聊天界面
15. 🧠 支持通过模型名称后缀设置 reasoning effort：
    1. OpenAI o系列模型
        - 添加后缀 `-high` 设置为 high reasoning effort (例如: `o3-mini-high`)
        - 添加后缀 `-medium` 设置为 medium reasoning effort (例如: `o3-mini-medium`)
        - 添加后缀 `-low` 设置为 low reasoning effort (例如: `o3-mini-low`)
    2. Claude 思考模型
        - 添加后缀 `-thinking` 启用思考模式 (例如: `claude-3-7-sonnet-20250219-thinking`)
16. 🔄 思考转内容功能
17. 🔄 针对用户的模型限流功能
18. 💰 缓存计费支持，开启后可以在缓存命中时按照设定的比例计费：
    1. 在 `系统设置-运营设置` 中设置 `提示缓存倍率` 选项
    2. 在渠道中设置 `提示缓存倍率`，范围 0-1，例如设置为 0.5 表示缓存命中时按照 50% 计费
    3. 支持的渠道：
        - [x] OpenAI
        - [x] Azure
        - [x] DeepSeek
        - [x] Claude

## 模型支持

此版本支持多种模型，详情请参考[接口文档-中继接口](https://docs.newapi.pro/api)：

1. 第三方模型 **gpts** （gpt-4-gizmo-*）
2. 第三方渠道[Midjourney-Proxy(Plus)](https://github.com/novicezk/midjourney-proxy)接口，[接口文档](https://docs.newapi.pro/api/midjourney-proxy-image)
3. 第三方渠道[Suno API](https://github.com/Suno-API/Suno-API)接口，[接口文档](https://docs.newapi.pro/api/suno-music)
4. 自定义渠道，支持填入完整调用地址
5. Rerank模型（[Cohere](https://cohere.ai/)和[Jina](https://jina.ai/)），[接口文档](https://docs.newapi.pro/api/jinaai-rerank)
6. Claude Messages 格式，[接口文档](https://docs.newapi.pro/api/anthropic-chat)
7. Dify，当前仅支持chatflow