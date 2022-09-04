package zjcgo

import (
	"encoding/base64"
	"net/http"
)

type Accounts struct {
	//自定义处理方式，如果认证失败
	UnAuthHandler func(ctx *Context)
	Users         map[string]string
}

func (a *Accounts) BasicAuth(next HandlerFunc) HandlerFunc {
	return func(ctx *Context) {
		//判断请求中是否有Authorization的Header
		username, password, ok := ctx.R.BasicAuth()
		if !ok {
			a.UnAuthHandlers(ctx)
			return
		}
		pw, ok := a.Users[username]
		//用户名
		if !ok {
			a.UnAuthHandlers(ctx)
			return
		}
		//密码错误
		if pw != password {
			a.UnAuthHandlers(ctx)
			return
		}
		ctx.Set("user", username)
		next(ctx)
	}
}
func (a *Accounts) UnAuthHandlers(ctx *Context) {
	//如果
	if a.UnAuthHandler != nil {
		a.UnAuthHandler(ctx)
	} else {
		ctx.W.WriteHeader(http.StatusUnauthorized) //未认证
	}
}
func BasicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
