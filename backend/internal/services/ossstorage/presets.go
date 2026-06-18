package ossstorage

import "strings"

type ProviderPreset struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	EndpointHint string `json:"endpoint_hint"`
	RegionHint   string `json:"region_hint"`
	UsePathStyle bool   `json:"use_path_style"`
	IsLocal      bool   `json:"is_local"`
}

var ProviderPresets = []ProviderPreset{
	{Key: "local", Name: "本机存储", EndpointHint: "", RegionHint: "", IsLocal: true},
	{Key: "minio", Name: "MinIO / 自建 S3", EndpointHint: "http://127.0.0.1:9000", RegionHint: "us-east-1", UsePathStyle: true},
	{Key: "aliyun", Name: "阿里云 OSS", EndpointHint: "https://oss-{region}.aliyuncs.com", RegionHint: "oss-cn-hangzhou", UsePathStyle: false},
	{Key: "tencent", Name: "腾讯云 COS", EndpointHint: "https://cos.{region}.myqcloud.com", RegionHint: "ap-guangzhou", UsePathStyle: false},
	{Key: "aws", Name: "Amazon S3", EndpointHint: "https://s3.{region}.amazonaws.com", RegionHint: "us-east-1", UsePathStyle: false},
	{Key: "google", Name: "Google Cloud Storage", EndpointHint: "https://storage.googleapis.com", RegionHint: "auto", UsePathStyle: false},
	{Key: "ibm", Name: "IBM Cloud Object Storage", EndpointHint: "https://s3.{region}.cloud-object-storage.appdomain.cloud", RegionHint: "us-south", UsePathStyle: true},
	{Key: "custom", Name: "自定义 S3 兼容", EndpointHint: "https://s3.example.com", RegionHint: "us-east-1", UsePathStyle: true},
}

func ResolveEndpoint(provider, region, endpoint string) string {
	if endpoint != "" {
		return endpoint
	}
	for _, p := range ProviderPresets {
		if p.Key != provider {
			continue
		}
		h := p.EndpointHint
		if strings.Contains(h, "{region}") && region != "" {
			return strings.ReplaceAll(h, "{region}", region)
		}
		return h
	}
	return endpoint
}

func DefaultPathStyle(provider string) bool {
	for _, p := range ProviderPresets {
		if p.Key == provider {
			return p.UsePathStyle
		}
	}
	return true
}
