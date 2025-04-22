package oss

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/url"
	"strings"
	"sync"
	"time"

	"io"

	"github.com/EICHI-X/ptools/logs"
	"github.com/EICHI-X/ptools/paerospike"
	"github.com/EICHI-X/ptools/purl"
	"github.com/EICHI-X/ptools/putils"
	"github.com/bytedance/sonic"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/nfnt/resize"
	"github.com/pkg/errors"
)

// url 作为唯一key
type Blog_file struct {
	Filename    string `gorm:"filename" json:"filename"`         // 文件名字
	ContentType string `gorm:"content_type" json:"content_type"` // 内容类型
	Owner       string `gorm:"owner" json:"owner"`               // 用户名
	Url         string `gorm:"url" json:"url"`                   // url
	Href        string `gorm:"href" json:"href"`                 // href
	Path        string `gorm:"path" json:"path"`                 // 路径
	Bucket      string `gorm:"bucket" json:"bucket"`             // Bucket
	Alt         string `gorm:"alt" json:"alt"`                   // alt
	Md5         string `gorm:"md5" json:"md5"`                   // md5
	Prefix      string `gorm:"prefix" json:"prefix"`             // 前缀
	MinioKey    string `gorm:"minio_key" json:"minio_key"`       // minio_key的key
	SaveType    string `gorm:"save_type" json:"save_type"`       // 存储类型
}

func (a *Blog_file) TableName() string {
	return "blog_file"
}

type OssLoader struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	Prefix          string

	FileUrl     string
	DefaultDir  string
	Host        string
	MinioClient *minio.Client
}
type ObjectEncodeType int

const (
	ObjectEncodeTypeDefault ObjectEncodeType = 0
)
const (
	UrlStart = "ostart:"
	UrlEnd   = ":oend"
)

type Object struct {
	Bucket           string                  `json:"bucket"`
	Object           string                  `json:"object"`
	ReqParams        *url.Values             `json:"-,omitempty"`
	PutObjectOptions *minio.PutObjectOptions `json:"-"`
	UploadInfo       *minio.UploadInfo       `json:"-"`
	UpdateCount      int                     `json:"update_count"`
	Host             string                  `json:"host"`
	SubHosts         []string                `json:"sub_hosts,omitempty"`
	SubRegion        []string                `json:"sub_regions,omitempty"`
	Region           string                  `json:"region"`
}

func NewObject(bucket string, object string) *Object {
	return &Object{
		Bucket: bucket,
		Object: object,
	}
}
func NewObjectWithParams(bucket string, object string, reqParams *url.Values, putOptions *minio.PutObjectOptions) *Object {
	return &Object{
		Bucket:           bucket,
		Object:           object,
		ReqParams:        reqParams,
		PutObjectOptions: putOptions,
	}
}
func (o *Object) EncodeUrl() string {
	s := purl.EncodeUrlToBase64(putils.ToJson(o))
	return UrlStart + s + UrlEnd
}
func IsOssUrlEncodedUrl(url string) bool {
	if (len(url) > len(UrlStart)+len(UrlEnd)) && url[:len(UrlStart)] == UrlStart && url[len(url)-len(UrlEnd):] == UrlEnd {
		return true //fmt.Errorf("url not valid %v", url)
	}
	return false
}
func isHttpUrl(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")

}
func DecodeUrlToObject(url string) (*Object, error) {
	v := Object{}
	if (len(url) < len(UrlStart)+len(UrlEnd)) || url[:len(UrlStart)] != UrlStart || url[len(url)-len(UrlEnd):] != UrlEnd {
		return &v, fmt.Errorf("url not valid %v", url)
	}
	url = url[len(UrlStart) : len(url)-len(UrlEnd)]
	urlDecode, err := purl.DecodeUrlFromBase64(url)
	if err == nil && len(url) > 0 {
		url = urlDecode
	}
	if err := sonic.Unmarshal([]byte(url), &v); err != nil {
		return &v, errors.WithMessage(err, "DecodeUrlToObject fail")

	}
	return &v, nil

}
func (o *Object) EncodeToHttps() string {
	if o.Host == "" {
		o.Host = ""
	}
	return fmt.Sprintf("https://%v/%v/%v", o.Host, o.Bucket, o.Object)
}
func (o *Object) DecodeHttps(url string) error {

	i := strings.Index(url, "https://")
	if i >= 0 {
		url = url[i+8:]
	} else {
		i = strings.Index(url, "http://")
		if i >= 0 {
			url = url[i+7:]
		}
	}
	if i < 0 {
		return fmt.Errorf("url not valid %v", url)
	}
	i = strings.Index(url, "/")
	if i < 0 {
		return fmt.Errorf("url not valid %v", url)
	}
	o.Host = url[:i]
	url = url[i+1:]
	i = strings.Index(url, "/")
	if i < 0 {
		return fmt.Errorf("url not valid %v", url)
	}
	o.Bucket = url[:i]
	o.Object = url[i:]
	return nil

}

// 根据 o.Host,o.Bucket,o.Object 拼接成key
func (o *Object) GenKey() string {
	k := "h%v.b%v.o%v"
	return fmt.Sprintf(k, o.Host, o.Bucket, o.Object)
}
func (o *Object) GetRealtimeUrl() string {

	return ""
}
func (o *Object) PresignedGetObject(ctx context.Context, client *OssLoader, expiry time.Duration, reqParams url.Values) (*url.URL, error) {
	return client.PresignedGetObject(ctx, o.Bucket, o.Object, expiry, reqParams)
}

func (o *OssLoader) GetRealUrlFromEncodedUrl(ctx context.Context, urlStr string, expiry time.Duration) (string, error) {
	if isHttpUrl(urlStr) {
		return urlStr, nil
	}
	if !IsOssUrlEncodedUrl(urlStr) {
		return "", fmt.Errorf("url is not valid %v", urlStr)
	}

	object, err := DecodeUrlToObject(urlStr)
	if err != nil {
		return "", err
	}
	var reqParams url.Values
	if object.ReqParams != nil {
		reqParams = *object.ReqParams
	}
	urlInfo, err := o.PresignedGetObject(ctx, object.Bucket, object.Object, expiry, reqParams)
	if err != nil || urlInfo == nil {
		return "", err
	}
	return urlInfo.String(), err
}
func getMinioClientWithRetry(o *OssLoader, retryTime int) (*minio.Client, error) {
	var minioClient *minio.Client
	var err error
	for retry := retryTime; retry > 0; retry-- {
		minioClient, err = minio.New(o.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(o.AccessKeyID, o.SecretAccessKey, ""),
			Secure: o.UseSSL,
		})
		if minioClient == nil || err != nil {
			continue
		}
		break
	}
	return minioClient, err
}
func (o *OssLoader) GetRealUrlsWithCache(ctx context.Context, urls []string, expiry time.Duration, retryTime int) ([]string, error) {
	defer putils.TimeCostWithMsg(ctx, fmt.Sprintf("GetRealUrlsWithCache target expiry  %v", expiry))()
	objs := make([]*Object, len(urls))
	keys := make([]string, len(urls))
	resUrls := make([]string, len(urls))
	keyPattern := "1000|packer|article.%v"
	for i, urlStr := range urls {
		if isHttpUrl(urlStr) || !IsOssUrlEncodedUrl(urlStr) {
			resUrls[i] = urlStr
			continue
		}
		object, err := DecodeUrlToObject(urlStr)
		if err != nil {
			continue
		}
		objs[i] = object
		keys[i] = fmt.Sprintf(keyPattern, object.GenKey())
	}
	cacheClient := paerospike.NewDefaultClient("aerospike.stock.packer")
	if cacheClient == nil {
		return resUrls, fmt.Errorf("paerospike client is nil")
	}
	cacheResp, err := cacheClient.GetBatch(keys, 100)
	emptyUrlObjs := make([]*Object, 0, len(urls))
	emptyObjIdxs := make([]int, 0, len(urls))
	if len(cacheResp) != len(urls) || err != nil {
		logs.CtxErrorf(ctx, "cacheResp len not match", putils.ToJsonSonic(urls))
	}
	for idx := range urls {
		urlStr := ""
		if idx < len(cacheResp) && len(cacheResp[idx]) > 0 {
			urlStr = cacheResp[idx]

		}
		// url 和 objs[idx] 不为空
		if len(urlStr) == 0 && objs[idx] != nil && len(urls[idx]) > 0 {
			emptyUrlObjs = append(emptyUrlObjs, objs[idx])
			emptyObjIdxs = append(emptyObjIdxs, idx)
		} else if len(urls[idx]) > 0 && len(urlStr) > 0 {
			resUrls[idx] = urlStr
		}

	}
	minioClient, err := getMinioClientWithRetry(o, 1)
	if minioClient == nil || err != nil {
		return cacheResp, err
	}
	wg := &sync.WaitGroup{}
	for i := range emptyUrlObjs {
		idx := i
		obj := emptyUrlObjs[i]
		if obj == nil || obj.Bucket == "" || obj.Object == "" {
			continue
		}
		wg.Add(1)
		go putils.GoFuncDone(ctx, wg, nil, func(ctx context.Context, param interface{}) {
			if retryTime <= 0 {
				retryTime = 1
			}
			for retry := retryTime; retry > 0; retry-- {
				if retry < retryTime {
					minioClient, err := getMinioClientWithRetry(o, 1)
					if minioClient == nil || err != nil {
						continue
					}
				}
				url, err := minioClient.PresignedGetObject(ctx, obj.Bucket, obj.Object, expiry, nil)
				if err != nil || url == nil {
					continue
				}
				resUrls[emptyObjIdxs[idx]] = url.String()
				// cache 的时间单位是秒，minio的时间单位是time.Duration,这里需要转换
				cacheExpiry := (expiry / 2) / time.Second // 缓存是真实事件的一半
				_ = cacheClient.PutAsync(fmt.Sprintf(keyPattern, obj.GenKey()), resUrls[emptyObjIdxs[idx]], uint32(cacheExpiry))
				break
			}
		})

	}
	wg.Wait()
	return resUrls, err
}
func NewOssLoader(endpoint string, AccessKeyID string, secretAccessKey string) (*OssLoader, error) {
	p := &OssLoader{
		Endpoint:        endpoint,
		AccessKeyID:     AccessKeyID,
		SecretAccessKey: secretAccessKey,
		UseSSL:          false,

		// Prefix: "stock",
		Host: endpoint,
	}
	minioClient, err := minio.New(p.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(p.AccessKeyID, p.SecretAccessKey, ""),
		Secure: p.UseSSL,
	})
	if err != nil {
		return nil, err
	}
	p.MinioClient = minioClient
	return p, nil
}

// 这里上传对外暴露的url是filename
func (u *OssLoader) UploadFromForm(ctx context.Context, bucket string, file multipart.File, fileObj *multipart.FileHeader, hashKey string) (string, string, error) {
	// 实现上传逻辑，返回文件路径与错误信息
	minioClient, err := minio.New(u.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(u.AccessKeyID, u.SecretAccessKey, ""),
		Secure: u.UseSSL,
	})
	if err != nil {
		// 如果minioClient创建失败，返回
		return "", "", err
	}
	// minio存储中的对象名称

	// 获取上传的文件句柄
	if err != nil {
		// 打开上传文件句柄失败，返回
		return "", "", err
	}
	// 调用Minio/ Sdk的对象上传

	info, err := minioClient.PutObject(ctx, bucket, fileObj.Filename, file, fileObj.Size, minio.PutObjectOptions{})
	if err != nil {
		// 对象上传失败，返回
		return "", "", err
	}
	url := fileObj.Filename
	return info.Key, url, nil
}
func (u *OssLoader) RemoveObject(bucket string, object string) {
	opts := minio.RemoveObjectOptions{}
	err := u.MinioClient.RemoveObject(context.Background(), bucket, object, opts)
	if err != nil {

		return
	}
}

// 这里上传对外暴露的url是filename
func (u *OssLoader) UploadFromFile(ctx context.Context, bucket string, file io.Reader, fileObj *multipart.FileHeader, isCheckEixst bool) (info *minio.UploadInfo, statInfo *minio.ObjectInfo, err error) {
	// 实现上传逻辑，返回文件路径与错误信息
	minioClient, err := minio.New(u.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(u.AccessKeyID, u.SecretAccessKey, ""),
		Secure: u.UseSSL,
	})
	info = &minio.UploadInfo{}
	if err != nil {
		// 如果minioClient创建失败，返回
		return
	}
	// minio存储中的对象名称
	stat, err := minioClient.StatObject(ctx, bucket, fileObj.Filename, minio.GetObjectOptions{})
	statInfo = &stat
	if err == nil && stat.Size > 0 {
		if isCheckEixst {
			info.Bucket = bucket
			info.Size = stat.Size
			return
		}
	}

	// 调用Minio/ Sdk的对象上传
	contentType := fileObj.Header.Get("Content-Type")
	putOption := minio.PutObjectOptions{}
	if len(contentType) > 0 {
		putOption.ContentType = contentType
	}

	infoUpload, err := minioClient.PutObject(ctx, bucket, fileObj.Filename, file, fileObj.Size, putOption)
	if err != nil {
		logs.CtxInfof(context.Background(), "upload fail %v", err)
		// 对象上传失败，返回
		return
	}
	info = &infoUpload

	return
}

// 这里上传对外暴露的url是filename
func (u *OssLoader) UploadImageWithMaxSize(ctx context.Context, bucket string, file io.Reader, fileObj *multipart.FileHeader, isCheckEixst bool, maxSize uint) (info *minio.UploadInfo, statInfo *minio.ObjectInfo, err error) {
	// 实现上传逻辑，返回文件路径与错误信息
	// 读取图片
	info = &minio.UploadInfo{}

	minioClient, err := minio.New(u.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(u.AccessKeyID, u.SecretAccessKey, ""),
		Secure: u.UseSSL,
	})
	if err != nil {
		// 如果minioClient创建失败，返回
		return
	}
	var stat minio.ObjectInfo

	stat, err = minioClient.StatObject(ctx, bucket, fileObj.Filename, minio.GetObjectOptions{})
	statInfo = &stat
	if err == nil && stat.Size > 0 {
		if isCheckEixst {
			info.Bucket = bucket
			info.Size = stat.Size
			return
		}
	}

	// 调用Minio/ Sdk的对象上传
	contentType := fileObj.Header.Get("Content-Type")
	putOption := minio.PutObjectOptions{}
	if len(contentType) > 0 {
		putOption.ContentType = contentType
	}
	var infoUpload minio.UploadInfo
	// minio存储中的对象名称
	fileSize := fileObj.Size
	if fileSize > int64(maxSize) {
		img, format, err1 := image.Decode(file)
		if err1 != nil {
			return info, nil, errors.WithMessage(err, "parse image fail")
		}

		// 关闭文件

		rate := float64(maxSize) / float64(fileSize)
		fileObj.Size = int64(maxSize)
		// 设置压缩后的宽度和高度，这里是压缩为原图宽度和高度的 1/4
		newWidth := uint(float64(img.Bounds().Dx()) * rate)
		newHeight := uint(float64(img.Bounds().Dy()) * rate)
		// 压缩图片
		resizedImg := resize.Resize(newWidth, newHeight, img, resize.Lanczos3)
		// 将压缩后的图片数据写入 bytes.Buffer
		var buf bytes.Buffer
		// 可以根据不同的格式执行相应的操作
		switch format {
		case "jpeg":
			err = jpeg.Encode(&buf, resizedImg, nil)
			if err != nil {
				return info, nil, err
			}
			// 处理 JPEG 格式的图像
		case "png":
			err = png.Encode(&buf, resizedImg)
			if err != nil {
				return info, nil, err
			}
			// 处理 PNG 格式的图像
		default:
			fmt.Println("Unsupported image format")
			return info, nil, errors.New("unsupported image format")
		}
		fileObj.Size = int64(buf.Len())
		fileObj.Header.Set("Content-Length", fmt.Sprintf("%v", fileObj.Size))
		infoUpload1, err2 := minioClient.PutObject(ctx, bucket, fileObj.Filename, &buf, fileObj.Size, putOption)
		if err2 != nil {
			logs.CtxInfof(context.Background(), "upload image fail %v", err)
			// 对象上传失败，返回
			return info, nil, err2
		}
		infoUpload = infoUpload1
		err = err2

	} else {
		infoUpload, err = minioClient.PutObject(ctx, bucket, fileObj.Filename, file, fileObj.Size, putOption)

	}
	if err != nil {
		logs.CtxInfof(context.Background(), "upload fail %v", err)
		// 对象上传失败，返回
		return info, nil, err
	}
	info = &infoUpload

	return
}
func (u *OssLoader) GetFileName(fileName string, hashKey string) string {
	objectName := "/"
	if u.Prefix != "" {
		objectName += u.Prefix
		objectName += "_"
	}
	// if hashKey != "" {

	// 	objectName += hashKey
	// 	objectName += "_"
	// }
	objectName += fileName
	return objectName
}
func (u *OssLoader) PresignedGetObject(ctx context.Context, bucket string, objectName string, expiry time.Duration, reqParams url.Values) (*url.URL, error) {
	minioClient, err := minio.New(u.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(u.AccessKeyID, u.SecretAccessKey, ""),
		Secure: u.UseSSL,
	})
	if minioClient == nil || err != nil {
		return nil, fmt.Errorf("OssLoader bucket %v client not exist", bucket)
	}
	url, err := minioClient.PresignedGetObject(ctx, bucket, objectName, expiry, nil)
	return url, err
}
func (u *OssLoader) DownLoadFile(ctx context.Context, bucket string, fileName string) (*minio.Object, error) {
	minioClient, err := minio.New(u.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(u.AccessKeyID, u.SecretAccessKey, ""),
		Secure: u.UseSSL,
	})
	if err != nil {
		return nil, err
	}
	object, err := minioClient.GetObject(ctx, bucket, fileName, minio.GetObjectOptions{})
	return object, err
}
func QueryFileInfoFromSlqUrl(ctx context.Context, url string) ([]*Blog_file, error) {
	var data = make([]*Blog_file, 0)

	return data, nil
}
func QueryFileInfoFromSlqHash(ctx context.Context, hash string) ([]*Blog_file, error) {

	var data = make([]*Blog_file, 0)

	return data, nil
}

func UpdateFileToSql(ctx context.Context, data *Blog_file) (*Blog_file, error) {

	return data, nil
}
