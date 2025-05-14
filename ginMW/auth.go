package ginMW

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/EICHI-X/ptools/logs"
	"github.com/EICHI-X/ptools/putils"
	"github.com/gin-gonic/gin"
)

const TokenUid = "token_uid"

type VerifyTokenResponse struct {
	Valid  bool   `json:"valid"`
	UserID int64  `json:"user_id"`
	Error  string `json:"error,omitempty"`
}

type CheckLoginOption struct {
	IsMustLogin  bool          // 未登录也可以进入页面，在函数内再校验
	IsNeedSetUid bool          // 是否需要设置uid到接口
	IsWithCors   bool          // 是否需要支持跨域
	Timeout      time.Duration // 超时时间
}

// 检查token是否为空
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

// 调用验证接口获取响应
func getVerifyTokenResp(ctx context.Context, url string, token string, timeout time.Duration) (*http.Response, error) {
	// 创建一个GET请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logs.CtxErrorf(ctx, "VerifyToken Error creating request: %v", err)
		return nil, err
	}

	// 设置Authorization头
	req.Header.Set("Authorization", token)

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
		logs.CtxErrorf(ctx, "VerifyToken Error sending request: %v", err)
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		return resp, nil
	}
	
	logs.CtxErrorf(ctx, "VerifyToken Error response: %v", putils.ToJson(resp))
	return resp, fmt.Errorf("Error response: %v %v", putils.ToJson(resp), err)
}

// CheckLoginMw Gin版的验证登录中间件
// url例子: http://your-api-host/api/v1/verify-token
// 校验 Authorization: Bearer token
func CheckLoginMw(url string, option CheckLoginOption) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if IsTokenEmpty(token) && option.IsMustLogin {
			c.JSON(http.StatusUnauthorized, gin.H{
				"valid": false,
				"error": "未登录",
			})
			c.Abort()
			return
		}
		
		logs.CtxInfof(c, "url: %v token: %v", url, token)
		
		if !IsTokenEmpty(token) {
			resp, err := getVerifyTokenResp(c, url, token, option.Timeout)
			defer func() {
				if resp != nil && resp.Body != nil {
					resp.Body.Close()
				}
			}()
			
			// 如果uid不为空表明上一层中间件已经解析验证过token了
			if GetTokenUid(c) != "" {
				c.Next()
				return
			}
			
			if err != nil || resp == nil || resp.StatusCode != http.StatusOK {
				if option.IsMustLogin {
					c.JSON(http.StatusUnauthorized, gin.H{
						"valid": false,
						"error": "验证登录未通过",
					})
					c.Abort()
					return
				}
			} else {
				if option.IsNeedSetUid {
					resBody, _ := io.ReadAll(resp.Body)
					
					var verifyResponse VerifyTokenResponse
					if err := json.Unmarshal(resBody, &verifyResponse); err == nil && verifyResponse.Valid {
						SetTokenUid(c, fmt.Sprintf("%d", verifyResponse.UserID))
					} else if option.IsMustLogin {
						errMsg := "验证失败"
						if verifyResponse.Error != "" {
							errMsg = verifyResponse.Error
						}
						c.JSON(http.StatusUnauthorized, gin.H{
							"valid": false,
							"error": errMsg,
						})
						c.Abort()
						return
					}
				}
			}
		}
		
		c.Next()
	}
}

// SetTokenUid 设置用户ID到上下文
func SetTokenUid(c *gin.Context, uid string) {
	c.Set(TokenUid, uid)
}

// GetTokenUid 从上下文获取用户ID
func GetTokenUid(c *gin.Context) string {
	uid := ""
	
	// 从gin.Context中获取
	if value, exists := c.Get(TokenUid); exists && value != nil {
		if uidStr, ok := value.(string); ok {
			uid = uidStr
		}
	}
	
	return uid
}

// GetTokenUidInt64 获取int64类型的用户ID
func GetTokenUidInt64(c *gin.Context) int64 {
	tokenUid := GetTokenUid(c)
	if tokenUid == "" {
		return 0
	}
	userId := putils.StrToInt64(tokenUid)
	return userId
}

// IsUidMatchTokenUid 判断uid是否与当前登录用户匹配
func IsUidMatchTokenUid(c *gin.Context, uid string) bool {
	tokenUid := GetTokenUid(c)
	return tokenUid == uid
}

// CorsMiddleware 跨域中间件
func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
} 
