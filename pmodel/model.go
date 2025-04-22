package pmodel

import (
	"context"

	"github.com/bytedance/gopkg/cloud/metainfo"
)

const ()

type TraceIDKeyType int

const (
	TraceIDKey TraceIDKeyType = iota
)
const TraceIdLogKey = "trace_id"

type EnvType string

const (
	EnvTypeProd EnvType = "prod"
	EnvTypePpe          = "ppe"
	EnvTypeBoe  EnvType = "boe"
)

type PassHeader struct {
	Env EnvType
}

var PassHeaderKeys = []string{
	"x-env",
	"x-env-type",
	"Authorization",
	"HUid",
	"HVersionCode",
	"HAppId",
	"HDeviceId",
}

type CommonHeader struct {
	Uid         string `json:"uid" gorm:"uid" form:"uid" dataframe:"uid" yaml:"uid"`
	VersionCode string `json:"version_code" gorm:"version_code" form:"version_code" dataframe:"version_code" yaml:"version_code"`
	AppId       string `json:"app_id" gorm:"app_id" form:"app_id" dataframe:"app_id" yaml:"app_id"`
	DeviceId    string `json:"device_id" gorm:"device_id" form:"device_id" dataframe:"device_id" yaml:"device_id"`
}

var EmptyStrs = []string{
	"",
	"null",
	"NULL",
	"none",
	"nil",
}

func IsEmptyString(s string) bool {
	for _, v := range EmptyStrs {
		if s == v {
			return true
		}
	}

	return false
}

// frameWork = hertz or kitex
func GetCommonHeader(ctx context.Context) *CommonHeader {
	r := &CommonHeader{}
	
	// token 中的 uid 优先级高于 HUid
	if v := ctx.Value("token_uid"); v != nil {
		if vStr, ok := v.(string); ok && !IsEmptyString(vStr) {
			r.Uid = vStr
		}
	}
	if r.Uid == "" {
		if v, ok := metainfo.GetPersistentValue(ctx, "HUid"); ok && !IsEmptyString(v) {
			r.Uid = v
		}
	}
	if v, ok := metainfo.GetPersistentValue(ctx, "HVersionCode"); ok && !IsEmptyString(v) {
		r.VersionCode = v
	}
	if v, ok := metainfo.GetPersistentValue(ctx, "HAppId"); ok && !IsEmptyString(v) {
		r.AppId = v
	}
	if v, ok := metainfo.GetPersistentValue(ctx, "HDeviceId"); ok && !IsEmptyString(v) {
		r.DeviceId = v
	}
	return r
}

// user_type=0 表示device_id, user_type=1 表示uid
func GetUserTypeAndId(ctx context.Context, uid string, isMustMatchUid bool) (int, string) {
	commonInfo := GetCommonHeader(ctx)
	if isMustMatchUid {
		if commonInfo.Uid != "" && commonInfo.Uid == uid {
			return 1, commonInfo.Uid
		}
	} else {
		if commonInfo.Uid != "" {
			return 1, commonInfo.Uid
		}
	}

	return 0, commonInfo.DeviceId
}
