package purl

import (
	"encoding/base64"
	"strings"
)
const (
    UrlListStep = ";"
)
func EncodeUrlToBase64(url string) string {
	encode := base64.URLEncoding.EncodeToString([]byte(url))
	return encode

}
func DecodeUrlFromBase64(urlEncoded string) (string ,error) {
	decodeUrl, err := base64.URLEncoding.DecodeString(urlEncoded)
	return string(decodeUrl),err

}
func EncodeUrlToBase64s(url []string) []string {
	    v := make([]string,len(url))
    for i := 0; i < len(url); i++ {
        encode := base64.URLEncoding.EncodeToString([]byte(url[i]))
        v[i] = encode
    }
    return v

}
func DecodeUrlFromBase64s(urlEncoded []string) ([]string ,[]error){
    v := make([]string,len(urlEncoded))
    e := make([]error,len(urlEncoded))
    for i := 0; i < len(urlEncoded); i++ {
        decodeUrl, err := base64.URLEncoding.DecodeString(urlEncoded[i])
        if err != nil {
            e[i] = err
        }
        v[i] = string(decodeUrl)
    }
    return v,e

}
func EncodeUrlsToStr(urls []string) string{
    d := EncodeUrlToBase64s(urls)
    return strings.Join(d,UrlListStep)

}
func DecodeStrToUrls(str string) []string{
    d := strings.Split(str,UrlListStep)
    v,_ := DecodeUrlFromBase64s(d)
    v1 := make([]string, 0)
    for idx := range v{
        if v[idx] != ""{
            v1 = append(v1, v[idx])
        }
    }
    return v1

}
