package hertzmiddleware

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"net/http"

	"fmt"

	"github.com/EICHI-X/ptools/logs"
	"github.com/EICHI-X/ptools/pmodel"
	"github.com/EICHI-X/ptools/putils"
	"github.com/bytedance/gopkg/cloud/metainfo"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/protocol"
	"go.opentelemetry.io/otel/trace"
)

func HertzClientAddTraceAndAuth(ctx context.Context, req *protocol.Request, resp *protocol.Response) (err error) {
	for idx := range pmodel.PassHeaderKeys {
		k := pmodel.PassHeaderKeys[idx]
		v, ok := metainfo.GetPersistentValue(ctx, k)
		if ok {
			req.SetHeader(k, v)

		}

	}
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().TraceID().IsValid() {
		req.SetHeader(RequestIDHeaderValue, span.SpanContext().TraceID().String())
	}
	return nil
}
func HertzClientTraceAndAuthMw(end client.Endpoint) client.Endpoint {

	return HertzClientAddTraceAndAuth
}

func GetClientPassHeader(ctx context.Context) http.Header {
	header := http.Header{}
	for idx := range pmodel.PassHeaderKeys {
		k := pmodel.PassHeaderKeys[idx]
		v, ok := metainfo.GetPersistentValue(ctx, k)
		if ok {
			header.Add(k, v)

		}

	}
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().TraceID().IsValid() {
		header.Add(RequestIDHeaderValue, span.SpanContext().TraceID().String())
	}
	return header
}

func GetHertzFromHertzHeader(ctx context.Context, c *app.RequestContext) http.Header {
	header := http.Header{}
	for idx := range pmodel.PassHeaderKeys {
		k := pmodel.PassHeaderKeys[idx]
		v := string(c.GetHeader(k))
		if len(v) > 0 {
			header.Add(k, v)
		}

	}
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().TraceID().IsValid() {
		header.Add(RequestIDHeaderValue, span.SpanContext().TraceID().String())
	}
	return header
}

type RequestFrom string

const (
	Hertz RequestFrom = "hertz"
	Kitex RequestFrom = "kitex"
)

// getLoginResp 获取token 检查用户是否登录
// https://user.huanfangsk.com//wuser/auth/check_login
func getLoginResp(ctx context.Context, url string, params map[string]string, requestFrom RequestFrom, timeout time.Duration) (*http.Response, error) {

	// 创建一个GET请求，并在URL中添加uid参数
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logs.CtxErrorf(ctx, " CheckLogin Error creating request:%v", err)
		return nil, err
	}
	if requestFrom == Kitex {
		headerKitex := GetClientPassHeader(ctx)
		for k := range headerKitex {
			req.Header.Set(k, headerKitex.Get(k))
		}
	} else {
		headerHz := GetClientPassHeader(ctx)
		for k := range headerHz {
			req.Header.Set(k, headerHz.Get(k))
		}
	}

	// 在URL中添加uid参数
	q := req.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}

	req.URL.RawQuery = q.Encode()
	timeOut := timeout
	if timeOut == 0 {
		timeOut = 3 * time.Second
	}
	// 发送请求
	client := &http.Client{
		Timeout: timeOut,
	}
	resp, err := client.Do(req)
	if err != nil {
		logs.CtxErrorf(ctx, " CheckLogin Error sending request:%v", err)

		return nil, err
	}

	if resp.StatusCode == http.StatusOK {

		return resp, nil
	}
	logs.CtxErrorf(ctx, " CheckLogin Error response :%v", putils.ToJson(resp))
	return resp, fmt.Errorf("Error response : %v %v", putils.ToJson(resp), err)

}

type CheckLoginResponse struct {
	//   1: required i32 status_code,
	// 2: optional string uid,
	// 3: optional string msg,

	StatusCode int32  `json:"status_code"`
	Uid        string `json:"uid"`
	Msg        string `json:"msg"`
}

const TokenUid = "token_uid"

type CheckLoginMwOption struct {
	IsMustLogin  bool          // 未登录也可以进入页面，在函数内再校验(只需要获取token uid的时候就为false)
	IsNeedSetUid bool          // 是否需要设置uid 到接口
	IsWithCors   bool          // 是否需要支持跨域
	Timeout      time.Duration // 超时时间
}

func IsTokenEmpty(token string) bool {
	if token == "" {
		return true
	}
	if token == "Bearer " {
		return true
	}
	if token == "Bearer" {
		return true
	}
	return false
}

// CheckLoginMw 获取token 检查用户是否登录
// url 例子: https://user.xxxx.com/wuser/auth/check_login,校验 Authorization:Bearer token
// 由于检测未登录会abort后面操作，在isWithCors为true时，需要在CheckLoginMw中加入CorsAllMiddleware，确保支持跨域
// isMustLogin 是否必须登录(只需要获取token uid的时候就为false)
func CheckLoginMw(url string, option CheckLoginMwOption) []app.HandlerFunc {
	mw := []app.HandlerFunc{func(c context.Context, ctx *app.RequestContext) {
		token := string(ctx.GetHeader("Authorization"))
		if IsTokenEmpty(token) && option.IsMustLogin {
			ctx.JSON(http.StatusUnauthorized, "CheckLogin fail")
			ctx.AbortWithMsg("未登录", http.StatusUnauthorized)
		}
		logs.CtxInfof(c, "ur; %v token: %v, ", url, token)
		if !IsTokenEmpty(token) {
			resp, err := getLoginResp(c, url, nil, Hertz, option.Timeout)
			defer func() {
				if resp != nil && resp.Body != nil {
					resp.Body.Close()
				}
			}()
			// 如果uid 不为空表面上一层中间件已经解析验证过token了
			if GetTokenUid(c, nil) != "" {
				ctx.Next(c)
				return
			}
			if err != nil || resp == nil || resp.StatusCode != http.StatusOK {
				if option.IsMustLogin {
					ctx.JSON(http.StatusUnauthorized, "CheckLogin fail")
					ctx.AbortWithMsg("验证登录未通过", http.StatusUnauthorized)
				}

			} else {
				if option.IsNeedSetUid {
					resBody, _ := io.ReadAll(resp.Body)

					var checkLoginResponse CheckLoginResponse
					if err := json.Unmarshal(resBody, &checkLoginResponse); err == nil {
						c = SetTokenUid(c, checkLoginResponse.Uid, ctx)
					}
				}
			}
		}

		ctx.Next(c)
	}}
	if option.IsWithCors {
		mw = append(mw, CorsAllMiddleware())
	}

	return mw
}

// 必须配合 CheckLoginMw  使用。
func SetTokenUid(ctx context.Context, uid string, c *app.RequestContext) context.Context {
	if c != nil {
		c.Set(TokenUid, uid)
	}
	return context.WithValue(ctx, TokenUid, uid)

}

// 第一个参数为context，第二个参数为app.RequestContext(如果有的话)否则填nil
func GetTokenUid(ctx context.Context, appCtx interface{}) string {
	uid := ""
	// 如果ctx中的uid不为空，尝试从ctx中获取，一般设置在非登录服务存在ctx中，在登录的服务设置在ctx
	tokenUid := ctx.Value(TokenUid)
	if tokenUid != nil {
		if v, ok := tokenUid.(string); ok {
			uid = v
		}
	}
	if uid != "" {
		return uid
	}
	// 如果ctx中为空，从app.RequestContext中获取
	if appCtx != nil {
		if appCtx, ok := appCtx.(*app.RequestContext); ok && appCtx != nil {
			if uidApp, okToken := appCtx.Get(TokenUid); okToken && uidApp != nil {
				if uidStr, ok := uidApp.(string); ok {
					uid = uidStr
				}
			}
		}
	}
	return uid

}
func GetTokenUidInt64(ctx context.Context, appCtx interface{}) int64 {
	tokenUid := GetTokenUid(ctx, appCtx)
	if tokenUid == "" {
		return 0
	}
	userId := putils.StrToInt64(tokenUid)
	return userId

}
func IsUidMatchTokenUid(c context.Context, uid string, appCtx interface{}) bool {
	tokenUid := GetTokenUid(c, appCtx)
	return tokenUid == uid
}
