<p align="right">
   <a href="./README.md">中文</a> | <strong>English</strong>
</p>
<div align="center">

![clinx](/web/public/logo.png)

# Clinx

🍥 Next-Generation Large Model Gateway and AI Asset Management System

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

## 📝 Project Description

> [!NOTE]  
> This project has undergone secondary development based on [New API](https://github.com/QuantumNous/new-api)

> [!IMPORTANT]  
> - The current version is for internal use by Clinx.
> - According to the [《Interim Measures for the Management of Generative Artificial Intelligence Services》](http://www.cac.gov.cn/2023-07/13/c_1690898327029107.htm), please do not provide any unregistered generative AI services to the public in China.

## 📚 Documentation

For detailed documentation, please visit: [https://dev.clinx.work/swag.html](https://dev.clinx.work/swag.html)

## 🏗 Architecture
![clinx-arch](/web/public/arch.png)

## ✨ Key Features

New API offers a wide range of features, please refer to [Features Introduction](https://docs.newapi.pro/wiki/features-introduction) for details:

1. 🎨 Brand new UI interface
2. 🌍 Multi-language support
3. 💰 Online recharge functionality (YiPay)
4. 🔍 Support for querying usage quotas with keys (works with [neko-api-key-tool](https://github.com/Calcium-Ion/neko-api-key-tool))
5. 🔄 Compatible with the original One API database
6. 💵 Support for pay-per-use model pricing
7. ⚖️ Support for weighted random channel selection
8. 📈 Data dashboard (console)
9. 🔒 Token grouping and model restrictions
10. 🤖 Support for more authorization login methods (LinuxDO, Telegram, OIDC)
11. 🔄 Support for Rerank models (Cohere and Jina), [API Documentation](https://docs.newapi.pro/api/jinaai-rerank)
12. ⚡ Support for OpenAI Realtime API (including Azure channels), [API Documentation](https://docs.newapi.pro/api/openai-realtime)
13. ⚡ Support for Claude Messages format, [API Documentation](https://docs.newapi.pro/api/anthropic-chat)
14. Support for entering chat interface via /chat2link route
15. 🧠 Support for setting reasoning effort through model name suffixes:
    1. OpenAI o-series models
        - Add `-high` suffix for high reasoning effort (e.g.: `o3-mini-high`)
        - Add `-medium` suffix for medium reasoning effort (e.g.: `o3-mini-medium`)
        - Add `-low` suffix for low reasoning effort (e.g.: `o3-mini-low`)
    2. Claude thinking models
        - Add `-thinking` suffix to enable thinking mode (e.g.: `claude-3-7-sonnet-20250219-thinking`)
16. 🔄 Thinking-to-content functionality
17. 🔄 Model rate limiting for users
18. 💰 Cache billing support, which allows billing at a set ratio when cache is hit:
    1. Set the `Prompt Cache Ratio` option in `System Settings-Operation Settings`
    2. Set `Prompt Cache Ratio` in the channel, range 0-1, e.g., setting to 0.5 means billing at 50% when cache is hit
    3. Supported channels:
        - [x] OpenAI
        - [x] Azure
        - [x] DeepSeek
        - [x] Claude

## Model Support

This version supports multiple models, please refer to [API Documentation-Relay Interface](https://docs.newapi.pro/api) for details:

1. Third-party models **gpts** (gpt-4-gizmo-*)
2. Third-party channel [Midjourney-Proxy(Plus)](https://github.com/novicezk/midjourney-proxy) interface, [API Documentation](https://docs.newapi.pro/api/midjourney-proxy-image)
3. Third-party channel [Suno API](https://github.com/Suno-API/Suno-API) interface, [API Documentation](https://docs.newapi.pro/api/suno-music)
4. Custom channels, supporting full call address input
5. Rerank models ([Cohere](https://cohere.ai/) and [Jina](https://jina.ai/)), [API Documentation](https://docs.newapi.pro/api/jinaai-rerank)
6. Claude Messages format, [API Documentation](https://docs.newapi.pro/api/anthropic-chat)
7. Dify, currently only supports chatflow