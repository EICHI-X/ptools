package env

import (
	"fmt"
	"net"
	"os"
	"strings"
)

type EnvType string

const (
	Prod = "prod"
	Ppe  = "ppe"
	Boe  = "boe"
)

type EnvStruct struct {
	EnvType     EnvType
	ServiceName string
	Env         string
}

var envInstance *EnvStruct

const (
	EnvTypeKey = "ENV_TYPE"
	EnvKey     = "ENV"
)

func IsProd() bool {
	return envInstance.EnvType == Prod
}
func IsPpe() bool {
	return envInstance.EnvType == Ppe
}
func IsBoe() bool {
	return envInstance.EnvType == Boe
}
func Instance() *EnvStruct {
	return envInstance
}
func init() {
	envInstance = &EnvStruct{}
	if c, ok := os.LookupEnv(EnvTypeKey); ok {
		envInstance.EnvType = EnvType(c)
	}
	if c, ok := os.LookupEnv(EnvKey); ok {
		envInstance.Env = c
	}
	if name, ok := os.LookupEnv("MY_SERVICE_NAME"); ok {
		envInstance.ServiceName = name
	}

}
func ResolvePsmToDns(psm string)(string,error){
	p := strings.Split(psm, ".")
	if len(p) != 3{
		return "",fmt.Errorf("psm:%v parse err,use like wealth.stock.mainstore", psm)
	}
	l := p[0]
	p[0] = p[3]
	p[3] = l
	return strings.Join(p, "."),nil
}
func ResolvePsmToIp(psm string)(string,error){
	p := strings.Split(psm, ".")
	if len(p) != 3{
		return "",fmt.Errorf("psm:%v parse err,use like wealth.stock.mainstore", psm)
	}
	l := p[0]
	p[0] = p[2]
	p[2] = l
	
	dns := strings.Join(p, ".")
	 ip,err :=  net.ResolveIPAddr("ip", dns)
   if err != nil || ip == nil{
        return "", fmt.Errorf("parse psm:%v fail ip=%v,err=%v",psm,ip,err)
   }
   return ip.IP.String(),nil
}
