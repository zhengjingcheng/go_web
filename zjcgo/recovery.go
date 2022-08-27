package zjcgo

import (
	"errors"
	"fmt"
	"github.com/zhengjingcheng/zjcgo/mserror"
	"net/http"
	"runtime"
	"strings"
)

func detailMsg(err any) string {
	var sb strings.Builder
	var pcs = make([]uintptr, 32)
	n := runtime.Callers(3, pcs)
	sb.WriteString(fmt.Sprintf("%v\n", err))
	for _, pc := range pcs[:n] {
		//函数
		fn := runtime.FuncForPC(pc)
		file, line := fn.FileLine(pc)
		sb.WriteString(fmt.Sprintf("\n\t%s:%d", file, line))
	}
	return sb.String()
}
func Recovery(next HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		defer func() {
			if err := recover(); err != nil {
				if e := err.(error); e != nil {
					var msError *mserror.MsError
					if errors.As(e, &msError) {
						msError.ExecResult()
						return
					}
				}
				ctx.Logger.Error(detailMsg(err))
				ctx.Fail(http.StatusInternalServerError, "Internal Server Error")
			}
		}()
		next(ctx)
	}
}
