package putils

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/EICHI-X/ptools/logs"
)

// GoFuncDone 封装包含了recover() 和 wg.Done()
func GoFuncDone(ctx context.Context, wg *sync.WaitGroup, param interface{}, f func(ctx context.Context, param interface{})) {
	defer func() {
		wg.Done()
	}()
	defer func() {
		if r := recover(); r != nil {
			// 打印错误栈
			stack := string(debug.Stack())
			// 使用fmt.Print 打印到run.log
			// fmt.Printf("[PanicHandler] %v", value)
			fmt.Print(stack)
			logs.CtxDebugf(ctx, "handle panic %v", stack)
		}
	}()
	f(ctx, param)

}
