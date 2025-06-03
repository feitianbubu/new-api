#!/bin/bash
VERSION=$1
swag init --generatedTime --parseDependency --ot=json -o=web/dist/swag --md=docs/api-descriptions
sed -i "s/\"version\": \".*\"/\"version\": \"$VERSION\"/" web/dist/swag/swagger.json
echo "Adding image analysis API to swagger..."
#sed -i '/^        "\/api\/v1\/chat\/completions": {$/,/^        },$/{
#    /^        },$/a\
#        "/api/v1/chat/completions": {\
#            "post": {\
#                "description": "接收符合 OpenAI API 格式的图片分析请求",\
#                "consumes": [\
#                    "application/json"\
#                ],\
#                "produces": [\
#                    "application/json",\
#                    "text/event-stream"\
#                ],\
#                "tags": [\
#                    "Clinx"\
#                ],\
#                "summary": "图片理解",\
#                "parameters": [\
#                    {\
#                        "type": "string",\
#                        "example": "Bearer sk-t8uP8tR6EhrmVgTijsf5HzMrr5KGE0BYCFTtSh4sk2GCXNZN",\
#                        "description": "用户认证令牌 (Bearer sk-xxxx)",\
#                        "name": "Authorization",\
#                        "in": "header",\
#                        "required": true\
#                    },\
#                    {\
#                        "description": "OpenAI 请求体",\
#                        "name": "request",\
#                        "in": "body",\
#                        "required": true,\
#                        "schema": {\
#                            "$ref": "#/definitions/dto.ExampleImageAnalysisRequest"\
#                        }\
#                    }\
#                ],\
#                "responses": {\
#                    "200": {\
#                        "description": "OK",\
#                        "schema": {\
#                            "$ref": "#/definitions/dto.OpenAITextResponse"\
#                        }\
#                    }\
#                }\
#            }\
#        },
#}' web/dist/swag/swagger.json
#
#sed -i '/^        "dto.ExampleGeneralOpenAIRequest": {$/,/^        },$/{
#    /^        },$/a\
#        "dto.ExampleImageAnalysisRequest": {\
#            "type": "object",\
#            "properties": {\
#                "messages": {\
#                    "type": "array",\
#                    "example": [{\
#                        "role": "user",\
#                        "content": [\
#                            {\
#                                "type": "text",\
#                                "text": "这张图片里有什么？"\
#                            },\
#                            {\
#                                "type": "image_url",\
#                                "image_url": {\
#                                    "url": "https://upload.wikimedia.org/wikipedia/commons/thumb/d/dd/Gfp-wisconsin-madison-the-nature-boardwalk.jpg/2560px-Gfp-wisconsin-madison-the-nature-boardwalk.jpg"\
#                                }\
#                            }\
#                        ]\
#                    }]\
#                },\
#                "model": {\
#                    "type": "string",\
#                    "example": "gpt-4.1"\
#                },\
#                "max_tokens": {\
#                    "type": "integer",\
#                    "example": 300\
#                }\
#            }\
#        },
#}' web/dist/swag/swagger.json