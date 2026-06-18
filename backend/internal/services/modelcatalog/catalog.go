package modelcatalog

// Modality constants for catalog grouping.
const (
	ModalityText  = "text"
	ModalityImage = "image"
	ModalityAudio = "audio"
	ModalityVideo = "video"
	ModalityVision = "vision"
)

// DeployVia indicates how the panel can run this model.
const (
	DeployViaTGI      = "tgi"
	DeployViaOllama   = "ollama"
	DeployViaSDWebUI  = "sd-webui"
	DeployViaComfyUI  = "comfyui"
	DeployViaWhisper  = "whisper"
	DeployViaManual   = "manual"
)

// HubTask describes a Hugging Face Hub pipeline filter.
type HubTask struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	LabelEN     string `json:"label_en"`
	Modality    string `json:"modality"`
	Placeholder string `json:"placeholder"`
}

func HubTasks() []HubTask {
	return []HubTask{
		{ID: "text-generation", Label: "文本生成 / 对话", LabelEN: "Text generation", Modality: ModalityText, Placeholder: "Qwen, Llama, Mistral…"},
		{ID: "text-to-image", Label: "文生图", LabelEN: "Text to image", Modality: ModalityImage, Placeholder: "SDXL, FLUX, Stable Diffusion…"},
		{ID: "image-to-text", Label: "图像理解", LabelEN: "Image to text", Modality: ModalityVision, Placeholder: "BLIP, LLaVA, Qwen-VL…"},
		{ID: "automatic-speech-recognition", Label: "语音识别 ASR", LabelEN: "Speech recognition", Modality: ModalityAudio, Placeholder: "Whisper, Distil-Whisper…"},
		{ID: "text-to-speech", Label: "语音合成 TTS", LabelEN: "Text to speech", Modality: ModalityAudio, Placeholder: "Bark, XTTS, SpeechT5…"},
		{ID: "text-to-video", Label: "文生视频", LabelEN: "Text to video", Modality: ModalityVideo, Placeholder: "HunyuanVideo, LTX…"},
		{ID: "audio-to-audio", Label: "音频处理", LabelEN: "Audio processing", Modality: ModalityAudio, Placeholder: "MusicGen, Demucs…"},
		{ID: "all", Label: "全部类型", LabelEN: "All types", Modality: "", Placeholder: "搜索 huggingface.co 模型…"},
	}
}

// ModelCatalogEntry describes a deployable mainstream model preset.
type ModelCatalogEntry struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	HFModelID    string   `json:"hf_model_id"`
	OllamaModel  string   `json:"ollama_model"`
	Modality     string   `json:"modality"`
	PipelineTag  string   `json:"pipeline_tag"`
	DeployVia    string   `json:"deploy_via"`
	AppStoreKey  string   `json:"app_store_key"`
	Category     string   `json:"category"`
	Params       string   `json:"params"`
	SizeHint     string   `json:"size_hint"`
	MinVRAMGB    int      `json:"min_vram_gb"`
	CPUOk        bool     `json:"cpu_ok"`
	Gated        bool     `json:"gated"`
	TGI          bool     `json:"tgi"`
	Ollama       bool     `json:"ollama"`
	HubDeployable bool    `json:"hub_deployable"`
	Tags         []string `json:"tags"`
	Description  string   `json:"description"`
	Featured     bool     `json:"featured"`
}

func Catalog() []ModelCatalogEntry {
	return append(textCatalog(), append(imageCatalog(), append(audioCatalog(), append(videoCatalog(), visionCatalog()...)...)...)...)
}

func textCatalog() []ModelCatalogEntry {
	return []ModelCatalogEntry{
		entry("qwen2.5-0.5b", "Qwen2.5 0.5B", "Qwen/Qwen2.5-0.5B-Instruct", "qwen2.5:0.5b", ModalityText, "text-generation", DeployViaTGI, "", "light", "0.5B", "~1 GB", 0, true, false, true, true, true, true,
			[]string{"中文", "轻量", "CPU"}, "无 GPU 首选，面板内置助手与 Web 对话均可流畅运行。"),
		entry("qwen2.5-1.5b", "Qwen2.5 1.5B", "Qwen/Qwen2.5-1.5B-Instruct", "qwen2.5:1.5b", ModalityText, "text-generation", DeployViaTGI, "", "light", "1.5B", "~3 GB", 0, true, false, true, true, true, true,
			[]string{"中文", "轻量"}, "比 0.5B 更聪明，CPU 可跑，适合 8GB 内存服务器。"),
		entry("qwen2.5-7b", "Qwen2.5 7B", "Qwen/Qwen2.5-7B-Instruct", "qwen2.5:7b", ModalityText, "text-generation", DeployViaTGI, "", "chat", "7B", "~5 GB", 8, false, false, true, true, true, true,
			[]string{"中文", "主流", "推荐"}, "国内最常用开源对话模型之一，需 8GB+ 显存。"),
		entry("llama3.2-3b", "Llama 3.2 3B", "meta-llama/Llama-3.2-3B-Instruct", "llama3.2:3b", ModalityText, "text-generation", DeployViaTGI, "", "chat", "3B", "~2 GB", 4, true, true, true, true, true, true,
			[]string{"Meta", "英文"}, "Meta 小模型，gated 需在 Hugging Face 申请 Token。"),
		entry("llama3.1-8b", "Llama 3.1 8B", "meta-llama/Llama-3.1-8B-Instruct", "llama3.1:8b", ModalityText, "text-generation", DeployViaTGI, "", "chat", "8B", "~5 GB", 8, false, true, true, true, true, true,
			[]string{"Meta", "主流"}, "国际主流 8B 对话模型。"),
		entry("mistral-7b", "Mistral 7B", "mistralai/Mistral-7B-Instruct-v0.3", "mistral:7b", ModalityText, "text-generation", DeployViaTGI, "", "chat", "7B", "~5 GB", 8, false, false, true, true, true, false,
			[]string{"欧洲", "主流"}, "Mistral 7B 指令微调版。"),
		entry("gemma2-2b", "Gemma 2 2B", "google/gemma-2-2b-it", "gemma2:2b", ModalityText, "text-generation", DeployViaTGI, "", "light", "2B", "~2 GB", 4, true, true, true, true, true, false,
			[]string{"Google"}, "Google Gemma 2 小模型。"),
		entry("phi3-mini", "Phi-3 Mini", "microsoft/Phi-3-mini-4k-instruct", "phi3:mini", ModalityText, "text-generation", DeployViaTGI, "", "chat", "3.8B", "~2.5 GB", 4, true, false, true, true, true, false,
			[]string{"Microsoft"}, "微软小参数模型，推理效率较高。"),
		entry("deepseek-r1-7b", "DeepSeek R1 7B", "", "deepseek-r1:7b", ModalityText, "text-generation", DeployViaOllama, "", "reasoning", "7B", "~5 GB", 8, false, false, false, true, true, true,
			[]string{"推理", "DeepSeek"}, "推理增强模型，推荐 Ollama 部署。"),
		entry("qwen2.5-coder-1.5b", "Qwen2.5 Coder 1.5B", "Qwen/Qwen2.5-Coder-1.5B-Instruct", "qwen2.5-coder:1.5b", ModalityText, "text-generation", DeployViaTGI, "", "code", "1.5B", "~3 GB", 0, true, false, true, true, true, true,
			[]string{"代码", "中文"}, "代码补全与解释。"),
		entry("codellama-7b", "Code Llama 7B", "codellama/CodeLlama-7b-Instruct-hf", "codellama:7b", ModalityText, "text-generation", DeployViaTGI, "", "code", "7B", "~4 GB", 8, false, false, true, true, true, false,
			[]string{"代码", "Meta"}, "Meta 代码模型。"),
		entry("smollm2-360m", "SmolLM2 360M", "HuggingFaceTB/SmolLM2-360M-Instruct", "", ModalityText, "text-generation", DeployViaTGI, "", "light", "360M", "~0.5 GB", 0, true, false, true, false, true, false,
			[]string{"极轻量"}, "最小可用模型，适合极低配测试。"),
	}
}

func imageCatalog() []ModelCatalogEntry {
	return []ModelCatalogEntry{
		entry("flux-schnell", "FLUX.1 Schnell", "black-forest-labs/FLUX.1-schnell", "", ModalityImage, "text-to-image", DeployViaComfyUI, "comfyui", "image", "12B", "~24 GB", 16, false, true, false, false, false, true,
			[]string{"文生图", "高质量", "推荐"}, "2024 主流文生图模型，需 ComfyUI / SD WebUI + GPU。"),
		entry("sdxl-base", "Stable Diffusion XL", "stabilityai/stable-diffusion-xl-base-1.0", "", ModalityImage, "text-to-image", DeployViaSDWebUI, "sd-webui", "image", "SDXL", "~7 GB", 8, false, false, false, false, false, true,
			[]string{"文生图", "Stable Diffusion"}, "最常用 SDXL 基座，适合 SD WebUI 一键绘图。"),
		entry("sd15", "Stable Diffusion 1.5", "runwayml/stable-diffusion-v1-5", "", ModalityImage, "text-to-image", DeployViaSDWebUI, "sd-webui", "image", "SD1.5", "~4 GB", 6, false, false, false, false, false, true,
			[]string{"文生图", "轻量"}, "经典 SD1.5，6GB 显存可跑，插件生态丰富。"),
		entry("sd-turbo", "SDXL Turbo", "stabilityai/sdxl-turbo", "", ModalityImage, "text-to-image", DeployViaSDWebUI, "sd-webui", "image", "SDXL", "~7 GB", 8, false, false, false, false, false, false,
			[]string{"文生图", "快速"}, "少步数快速出图。"),
		entry("sd3-medium", "Stable Diffusion 3", "stabilityai/stable-diffusion-3-medium-diffusers", "", ModalityImage, "text-to-image", DeployViaComfyUI, "comfyui", "image", "2B", "~10 GB", 12, false, true, false, false, false, false,
			[]string{"文生图", "SD3"}, "Stability SD3 Medium，需 ComfyUI 工作流。"),
		entry("playground-v2.5", "Playground v2.5", "playgroundai/playground-v2.5-1024px-aesthetic", "", ModalityImage, "text-to-image", DeployViaSDWebUI, "sd-webui", "image", "SDXL", "~7 GB", 8, false, false, false, false, false, false,
			[]string{"文生图", "美学"}, "偏美学风格的高质量文生图。"),
	}
}

func audioCatalog() []ModelCatalogEntry {
	return []ModelCatalogEntry{
		entry("whisper-large-v3", "Whisper Large v3", "openai/whisper-large-v3", "", ModalityAudio, "automatic-speech-recognition", DeployViaWhisper, "whisper", "asr", "Large", "~3 GB", 4, false, false, false, false, false, true,
			[]string{"语音识别", "OpenAI", "推荐"}, "最强开源 ASR，面板软件商店可装 Whisper。"),
		entry("whisper-small", "Whisper Small", "openai/whisper-small", "", ModalityAudio, "automatic-speech-recognition", DeployViaWhisper, "whisper", "asr", "Small", "~0.5 GB", 0, true, false, false, false, false, true,
			[]string{"语音识别", "CPU"}, "轻量语音识别，CPU 可跑。"),
		entry("distil-whisper", "Distil-Whisper Large", "distil-whisper/distil-large-v3", "", ModalityAudio, "automatic-speech-recognition", DeployViaWhisper, "whisper", "asr", "Large", "~1.5 GB", 4, false, false, false, false, false, false,
			[]string{"语音识别", "蒸馏"}, "Whisper 蒸馏版，速度更快。"),
		entry("bark", "Bark TTS", "suno/bark", "", ModalityAudio, "text-to-speech", DeployViaManual, "", "tts", "—", "~4 GB", 8, false, false, false, false, false, true,
			[]string{"语音合成", "多语言"}, "Suno Bark 文本转语音，含音效。"),
		entry("xtts-v2", "Coqui XTTS v2", "coqui/XTTS-v2", "", ModalityAudio, "text-to-speech", DeployViaManual, "", "tts", "—", "~2 GB", 6, false, false, false, false, false, false,
			[]string{"语音合成", "克隆"}, "支持声音克隆的多语言 TTS。"),
		entry("musicgen-small", "MusicGen Small", "facebook/musicgen-small", "", ModalityAudio, "audio-to-audio", DeployViaManual, "", "music", "300M", "~1 GB", 6, false, false, false, false, false, true,
			[]string{"音乐生成", "Meta"}, "Meta 文本/旋律生成音乐。"),
		entry("speecht5-tts", "SpeechT5 TTS", "microsoft/speecht5_tts", "", ModalityAudio, "text-to-speech", DeployViaManual, "", "tts", "—", "~0.5 GB", 4, true, false, false, false, false, false,
			[]string{"语音合成", "Microsoft"}, "微软 SpeechT5 语音合成。"),
	}
}

func videoCatalog() []ModelCatalogEntry {
	return []ModelCatalogEntry{
		entry("ltx-video", "LTX Video", "Lightricks/LTX-Video", "", ModalityVideo, "text-to-video", DeployViaManual, "", "video", "—", "~12 GB", 16, false, false, false, false, false, true,
			[]string{"文生视频", "推荐"}, "开源文生视频，需高显存 GPU 与专用推理框架。"),
		entry("hunyuanvideo", "HunyuanVideo", "tencent/HunyuanVideo", "", ModalityVideo, "text-to-video", DeployViaManual, "", "video", "—", "~40 GB", 24, false, true, false, false, false, true,
			[]string{"文生视频", "腾讯"}, "腾讯混元视频大模型，仅适合高配 GPU。"),
		entry("cogvideox-2b", "CogVideoX 2B", "THUDM/CogVideoX-2b", "", ModalityVideo, "text-to-video", DeployViaManual, "", "video", "2B", "~8 GB", 12, false, false, false, false, false, false,
			[]string{"文生视频", "智谱"}, "相对轻量的开源文生视频模型。"),
	}
}

func visionCatalog() []ModelCatalogEntry {
	return []ModelCatalogEntry{
		entry("qwen2-vl-2b", "Qwen2-VL 2B", "Qwen/Qwen2-VL-2B-Instruct", "", ModalityVision, "image-to-text", DeployViaTGI, "", "vision", "2B", "~5 GB", 8, false, false, true, false, false, true,
			[]string{"视觉", "中文", "推荐"}, "Qwen 视觉语言模型，可图文对话。"),
		entry("llava-1.5-7b", "LLaVA 1.5 7B", "llava-hf/llava-1.5-7b-hf", "", ModalityVision, "image-to-text", DeployViaTGI, "", "vision", "7B", "~14 GB", 12, false, false, true, false, true, true,
			[]string{"视觉", "LLaVA"}, "经典图文理解模型。"),
		entry("blip-large", "BLIP Caption", "Salesforce/blip-image-captioning-large", "", ModalityVision, "image-to-text", DeployViaManual, "", "vision", "—", "~1 GB", 4, false, false, false, false, false, false,
			[]string{"图像描述"}, "图像打 caption，适合批量标注。"),
		entry("florence-2-base", "Florence-2 Base", "microsoft/Florence-2-base", "", ModalityVision, "image-to-text", DeployViaManual, "", "vision", "0.23B", "~1 GB", 4, true, false, false, false, false, false,
			[]string{"视觉", "Microsoft"}, "微软多任务视觉基础模型。"),
	}
}

func entry(
	id, name, hf, ollama, modality, pipeline, deployVia, appKey, category, params, size string,
	minVRAM int, cpuOk, gated, tgi, ollamaOk, hubDeploy, featured bool,
	tags []string, desc string,
) ModelCatalogEntry {
	return ModelCatalogEntry{
		ID: id, Name: name, HFModelID: hf, OllamaModel: ollama,
		Modality: modality, PipelineTag: pipeline, DeployVia: deployVia, AppStoreKey: appKey,
		Category: category, Params: params, SizeHint: size, MinVRAMGB: minVRAM,
		CPUOk: cpuOk, Gated: gated, TGI: tgi, Ollama: ollamaOk, HubDeployable: hubDeploy,
		Tags: tags, Description: desc, Featured: featured,
	}
}

func DefaultHFModelIDs() []string {
	var ids []string
	for _, m := range Catalog() {
		if m.HFModelID != "" && m.Modality == ModalityText {
			ids = append(ids, m.HFModelID)
		}
	}
	return ids
}

func ResolveEntry(id string) *ModelCatalogEntry {
	for _, m := range Catalog() {
		if m.ID == id || m.HFModelID == id || m.OllamaModel == id {
			copy := m
			return &copy
		}
	}
	return nil
}

func CatalogByModality(modality string) []ModelCatalogEntry {
	if modality == "" || modality == "all" {
		return Catalog()
	}
	var out []ModelCatalogEntry
	for _, m := range Catalog() {
		if m.Modality == modality {
			out = append(out, m)
		}
	}
	return out
}
