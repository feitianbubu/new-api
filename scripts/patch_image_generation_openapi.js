const fs = require('fs');

if (process.argv.length < 3) {
  console.error('Usage: node patch_image_generation_openapi.js <openapi3.json>');
  process.exit(1);
}

const file = process.argv[2];
const doc = JSON.parse(fs.readFileSync(file, 'utf8'));

const imageGenerationExamples = {
  openai: {
    summary: "OpenAI DALL-E 中文提示词示例",
    value: {
      model: "dall-e-3",
      prompt: "可爱的中国小女孩在花园里玩耍",
      response_format: "url",
      size: "1024x1024",
      quality: "standard",
      n: 1
    }
  },
  gemini: {
    summary: "Gemini 中文提示词示例",
    value: {
      model: "nano-banana",
      prompt: "一只猫在钢琴上跳舞",
      response_format: "url",
      size: "1024x1024",
      n: 1
    }
  },
  jimeng: {
    summary: "即梦生图中文提示词示例",
    value: {
      model: "doubao-seedream-4-0-250828",
      prompt: "可爱的中国小女孩在花园里玩耍",
      image: ["https://ark-project.tos-cn-beijing.volces.com/doc_image/seedream4_imagesToimages_1.png"],
      watermark: false,
      extra_fields: {
        width: 768,
        height: 512,
        seed: -1,
        use_pre_llm: true,
        use_sr: true,
        logo_info: {
          add_logo: true,
          position: 1,
          opacity: 0.4
        }
      }
    }
  }
};

// 可能的图像生成路径
const imagePaths = [
  "/api/v1/images/generations",
  "/v1/images/generations",
];

let patchedPaths = [];

imagePaths.forEach(path => {
  try {
    if (doc.paths && doc.paths[path] && doc.paths[path].post) {
      if (doc.paths[path].post.requestBody && 
          doc.paths[path].post.requestBody.content && 
          doc.paths[path].post.requestBody.content["application/json"]) {
        
        doc.paths[path].post.requestBody.content["application/json"].examples = imageGenerationExamples;
        patchedPaths.push(path);
      }
    }
  } catch (e) {
    console.warn(`Failed to patch ${path}:`, e.message);
  }
});

if (patchedPaths.length > 0) {
  fs.writeFileSync(file, JSON.stringify(doc, null, 2));
  console.log(`Patched openapi3.json with image generation examples for paths: ${patchedPaths.join(', ')}`);
} else {
  console.log("No image generation paths found in openapi3.json. Available paths:");
  if (doc.paths) {
    Object.keys(doc.paths).forEach(path => {
      console.log(`  - ${path}`);
    });
  }
} 