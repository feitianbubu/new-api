# 模型调用

`/api/v1/chat/completions` 接口支持两种使用模式：**文本对话**和**图片理解**

## 📝 文本对话模式

用于普通的AI文本对话交互，支持多轮对话。

### 请求示例

```json
{
  "model": "gpt-4.1",
  "messages": [
    {
      "role": "user",
      "content": "你好，请介绍一下自己"
    }
  ],
  "temperature": 0.7,
  "stream": false
}
```

## 🖼️ 图片理解模式

用于分析图像内容，可以同时包含文本提问和图像URL。

### 图片理解示例

```json
{
  "model": "gpt-4.1",
  "messages": [
    {
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "这张图片里有什么？"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "https://www.baidu.com/img/PCtm_d9c8750bed0b3c7d089fa7d55720d6cf.png"
          }
        }
      ]
    }
  ],
  "max_tokens": 300
}
```

### Base64图片示例

```json
{
  "model": "gpt-4.1", 
  "messages": [
    {
      "role": "user",
      "content": [
        {
          "type": "text",
          "text": "描述这张图片的内容"
        },
        {
          "type": "image_url",
          "image_url": {
            "url": "data:image/png;base64,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"
          }
        }
      ]
    }
  ],
  "max_tokens": 400
}
```