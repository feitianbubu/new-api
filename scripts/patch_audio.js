const fs = require('fs');

if (process.argv.length < 3) {
  console.error('Usage: node patch_audio_openapi.js <openapi3.json>');
  process.exit(1);
}

const file = process.argv[2];
const doc = JSON.parse(fs.readFileSync(file, 'utf8'));

const audioSpeechExamples = {
  openai: {
    summary: "OpenAI TTS Speech Example",
    value: {
      model: "gpt-4o-mini-tts",
      input: "The quick brown fox jumped over the lazy dog.",
      voice: "alloy"
    }
  },
  doubao: {
    summary: "Doubao TTS Speech Example with Chinese Text",
    value: {
      input: "你是一个好孩子",
      model: "seed-tts-1.1",
      response_format: "mp3",
      speed: 1,
      voice: "zh_female_shuangkuaisisi_emo_v2_mars_bigtts",
      metadata: {
        audio: {
          emotion_scale: 5,
          enable_emotion: true,
          emotion: "happy"
        }
      }
    }
  },
  minimax: {
    summary: "Minimax TTS Speech Example with Chinese Text",
    value: {
      model: "speech-2.5-hd-preview",
      input: "今天是不是很开心呀，当然了！",
      voice: "male-qn-qingse",
      metadata: {
        stream: false,
        voice_setting: {
          voice_id: "male-qn-qingse",
          speed: 1,
          vol: 1,
          pitch: 0,
          emotion: "happy"
        },
        pronunciation_dict: {
          tone: ["处理/(chu3)(li3)", "危险/dangerous"]
        },
          audio_setting: {
          sample_rate: 32000,
          bitrate: 128000,
          format: "mp3",
          channel: 1
        },
        output_format:"url",
        subtitle_enable: false
      }
    }
  }
};

// Possible audio speech paths
const audioSpeechPaths = [
  "/api/v1/audio/speech",
  "/v1/audio/speech",
];

let patchedPaths = [];

audioSpeechPaths.forEach(path => {
  try {
    if (doc.paths && doc.paths[path] && doc.paths[path].post) {
      if (doc.paths[path].post.requestBody &&
          doc.paths[path].post.requestBody.content &&
          doc.paths[path].post.requestBody.content["application/json"]) {

        doc.paths[path].post.requestBody.content["application/json"].examples = audioSpeechExamples;
        patchedPaths.push(path);
      }
    }
  } catch (e) {
    console.warn(`Failed to patch ${path}:`, e.message);
  }
});

if (patchedPaths.length > 0) {
  fs.writeFileSync(file, JSON.stringify(doc, null, 2));
  console.log(`Patched openapi3.json with audio speech examples for paths: ${patchedPaths.join(', ')}`);
} else {
  console.log("No audio speech paths found in openapi3.json. Available paths:");
  if (doc.paths) {
    Object.keys(doc.paths).forEach(path => {
      console.log(`  - ${path}`);
    });
  }
}