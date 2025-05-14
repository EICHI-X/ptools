package purl

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/EICHI-X/ptools/logs"
)

// FileInfo 文件信息
type FileInfo struct {
	FileName     string `json:"file_name"`
	Size         int64  `json:"size"`
	URL          string `json:"url"`
	ContentType  string `json:"content_type"`
	ProviderName string `json:"provider_name"`
	UploadTime   string `json:"upload_time"`
	IsPublic     bool   `json:"is_public"` // 是否为公开文件
	FilePath     string `json:"file_path"` // endpoint后面的路径，如：aipurchase/public/test_public.txt
}
// URLProcessor SDK用于处理URL
type URLProcessor struct{}

// NewURLProcessor 创建URL处理器实例
func NewURLProcessor() *URLProcessor {
	return &URLProcessor{}
}
var UrlFIleProcessor *URLProcessor = NewURLProcessor()
// ProcessURL 处理URL
// 如果URL以http开头，则直接返回
// 如果URL是base64编码的元数据，则解析并返回其中的URL字段
func (p *URLProcessor) ProcessURL(ctx context.Context, input string) (string) {
	// 检查是否是http开头的URL
	if strings.HasPrefix(strings.ToLower(input), "http") {
		return input
	}

	// 尝试解析为base64编码的元数据
	fileInfo, err := p.decodeFileMetadata(input)
	if err != nil {
		logs.Error(ctx, "无法处理输入: 既不是HTTP URL也不是有效的元数据: %w", err)
		return ""
	}

	// 返回FileInfo中的URL字段
	return fileInfo.URL
}

// decodeFileMetadata 解析base64编码的文件元数据
func (p *URLProcessor) decodeFileMetadata(encodedData string) (*FileInfo, error) {
	// 解码base64
	jsonData, err := base64.StdEncoding.DecodeString(encodedData)
	if err != nil {
		return nil, fmt.Errorf("解码base64失败: %w", err)
	}

	// 反序列化JSON
	var fileInfo FileInfo
	err = json.Unmarshal(jsonData, &fileInfo)
	if err != nil {
		return nil, fmt.Errorf("反序列化JSON失败: %w", err)
	}

	return &fileInfo, nil
}
