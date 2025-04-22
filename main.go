package main

import (
	"context"

	"github.com/EICHI-X/ptools/logs"
	"github.com/EICHI-X/ptools/putils"
)

func main() {
	logs.CtxDebugf(context.TODO(), "init", putils.ToJson(1))
	logs.GetLogger().SetLevel(logs.LevelDebug)
}
