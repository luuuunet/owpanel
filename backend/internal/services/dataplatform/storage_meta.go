package dataplatform

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type BucketMeta struct {
	Name         string `json:"name"`
	ObjectCount  int    `json:"object_count"`
	TotalSize    int64  `json:"total_size"`
	TotalHuman   string `json:"total_human"`
	StorageClass string `json:"storage_class,omitempty"`
}

type StorageEngineMeta struct {
	Key       string       `json:"key"`
	Name      string       `json:"name"`
	Installed bool         `json:"installed"`
	Running   bool         `json:"running"`
	Endpoint  string       `json:"endpoint"`
	UseCase   string       `json:"use_case"`
	Buckets   []BucketMeta `json:"buckets,omitempty"`
	Message   string       `json:"message,omitempty"`
}

var storageEngines = []struct {
	Key, Name, UseCase, Container string
	APIPort                       int
}{
	{Key: "minio", Name: "MinIO", UseCase: "S3-compatible distributed object storage", Container: "owpanel-minio", APIPort: 9000},
	{Key: "ceph", Name: "Ceph RGW", UseCase: "Ceph object gateway metadata for cluster backup", Container: "owpanel-ceph-rgw", APIPort: 7480},
}

func (s *Service) StorageMetadata() []StorageEngineMeta {
	out := make([]StorageEngineMeta, 0, len(storageEngines))
	for _, e := range storageEngines {
		st := StorageEngineMeta{
			Key:      e.Key,
			Name:     e.Name,
			UseCase:  e.UseCase,
			Endpoint: fmt.Sprintf("http://127.0.0.1:%d", e.APIPort),
		}
		if s.appstore != nil {
			app, err := s.appstore.Get(e.Key)
			if err == nil && app.Installed {
				st.Installed = true
				live := s.appstore.LiveStatus(e.Key)
				st.Running = live == "running"
			}
		}
		if st.Running {
			ak, sk := readDockerCreds(s.dataDir, e.Key)
			if ak == "" {
				ak, sk = "admin", "openpanel123"
			}
			buckets, msg := listS3Buckets(st.Endpoint, ak, sk)
			st.Buckets = buckets
			st.Message = msg
		}
		out = append(out, st)
	}
	return out
}

func readDockerCreds(dataDir, key string) (string, string) {
	p := filepath.Join(dataDir, "docker-secrets", key+".env")
	body, err := os.ReadFile(p)
	if err != nil {
		return "", ""
	}
	var ak, sk string
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "MINIO_ROOT_USER=") || strings.HasPrefix(line, "AWS_ACCESS_KEY_ID=") {
			ak = strings.SplitN(line, "=", 2)[1]
		}
		if strings.HasPrefix(line, "MINIO_ROOT_PASSWORD=") || strings.HasPrefix(line, "AWS_SECRET_ACCESS_KEY=") {
			sk = strings.SplitN(line, "=", 2)[1]
		}
	}
	return ak, sk
}

func listS3Buckets(endpoint, ak, sk string) ([]BucketMeta, string) {
	cfg := aws.Config{
		Region:      "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider(ak, sk, ""),
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(strings.TrimRight(endpoint, "/"))
		o.UsePathStyle = true
	})
	ctx := context.Background()
	out, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err.Error()
	}
	var buckets []BucketMeta
	for _, b := range out.Buckets {
		name := aws.ToString(b.Name)
		meta := BucketMeta{Name: name}
		lo, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: aws.String(name)})
		if err == nil {
			meta.ObjectCount = len(lo.Contents)
			for _, obj := range lo.Contents {
				meta.TotalSize += aws.ToInt64(obj.Size)
			}
			meta.TotalHuman = humanSize(meta.TotalSize)
		}
		buckets = append(buckets, meta)
	}
	return buckets, ""
}
