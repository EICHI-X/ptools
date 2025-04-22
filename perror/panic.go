package perror

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/EICHI-X/ptools/logs"
)

func PanicHandle(ctx context.Context) {
	if r := recover(); r != nil {
		// 打印错误栈
		// const size = 64 << 10
		// buf := make([]byte, size)
		// buf = buf[:runtime.Stack(buf, false)]
		stack := string(debug.Stack())
		fmt.Print("[PanicHandler]")
		fmt.Print(stack)
		logs.CtxErrorf(ctx, "go routine panic, err=%v: %s ", r, stack)

	}
}
