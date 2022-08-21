package zjcgo

import (
	"fmt"
	"github.com/zhengjingcheng/zjcgo/render"
	"html/template"
	"log"
	"net/http"
	"sync"
)

/*
	这个文件是服务的主要内容所在，支持路由匹配，渲染，参数提取等功能
*/

const ANY = "ANY"

//处理函数的句柄
type HandlerFunc func(ctx *Context)

//传入一个句柄函数，然后经过中间件处理后再将这个句柄函数返回回去
type MiddlewareFunc func(hanlerFunc HandlerFunc) HandlerFunc

//定义路由结构
type router struct {
	groupName          string                                 //属于哪个路由组
	handlerMap         map[string]map[string]HandlerFunc      //处理路由的函数 k1路径 k2请求方法
	middlewaresFuncMap map[string]map[string][]MiddlewareFunc //路由级别中间件（重要）
	treeNode           *treeNode                              //匹配路由的前缀树
	middlewares        []MiddlewareFunc                       //通用中间件
}

//路由组结构
type routerGroup struct {
	groups []*router //由一个个路由组成
}

//添加新路由
func (r *routerGroup) Group(name string) *router {
	g := &router{
		groupName:          name, //路由组的名字
		handlerMap:         make(map[string]map[string]HandlerFunc, 0),
		middlewaresFuncMap: make(map[string]map[string][]MiddlewareFunc, 0),
		treeNode:           &treeNode{name: "/", child: make([]*treeNode, 0)},
	}
	r.groups = append(r.groups, g)
	return g
}

/*
·······················································中间件部分·························································
*/
//前置中间件
func (r *router) Use(middlewareFunc ...MiddlewareFunc) {
	r.middlewares = append(r.middlewares, middlewareFunc...)
}

func (r *router) methodHandle(name string, method string, h HandlerFunc, ctx *Context) {
	//组通用中间件
	if r.middlewares != nil {
		for _, midwareFunc := range r.middlewares {
			h = midwareFunc(h)
		}
	}
	//组路由级别
	middlewareFuncs := r.middlewaresFuncMap[name][method]
	if middlewareFuncs != nil {
		for _, midwareFunc := range middlewareFuncs {
			h = midwareFunc(h)
		}
	}
	h(ctx)
}

/*
·····················································处理路由请求的函数····················································
*/
//处理路由请求的函数
func (r *router) Handler(name string, method string, handler HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	//判断当前路径有没有路由处理函数，没有则创建一个新的
	_, ok := r.handlerMap[name]
	if !ok {
		r.handlerMap[name] = make(map[string]HandlerFunc)
		r.middlewaresFuncMap[name] = make(map[string][]MiddlewareFunc)
	}
	_, ok = r.handlerMap[name][method]
	if ok {
		panic("重复路由请求")
	}
	r.handlerMap[name][method] = handler
	r.middlewaresFuncMap[name][method] = append(r.middlewaresFuncMap[name][method], middlewareFunc...)
	r.treeNode.Put(name)
}
func (r *router) Any(name string, handle HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.Handler(name, ANY, handle, middlewareFunc...)
}
func (r *router) Get(name string, handle HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.Handler(name, http.MethodGet, handle, middlewareFunc...)
}
func (r *router) Post(name string, handle HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.Handler(name, http.MethodPost, handle, middlewareFunc...)
}
func (r *router) Put(name string, handle HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.Handler(name, http.MethodPut, handle, middlewareFunc...)
}
func (r *router) Deletet(name string, handle HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.Handler(name, http.MethodDelete, handle, middlewareFunc...)
}

func (r *router) Patch(name string, handle HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.Handler(name, http.MethodPatch, handle, middlewareFunc...)
}
func (r *router) Options(name string, handle HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.Handler(name, http.MethodOptions, handle, middlewareFunc...)
}

func (r *router) Head(name string, handle HandlerFunc, middlewareFunc ...MiddlewareFunc) {
	r.Handler(name, http.MethodHead, handle, middlewareFunc...)
}

/*
··························································封装服务器引擎·················································
*/

//路由服务引擎(封装一个路由组)
type Engine struct {
	routerGroup                   //路由组，必须品
	funcMap     template.FuncMap  //加载模板的句柄函数
	HTMLRender  render.HTMLRender //HTML渲染函数
	pool        sync.Pool         //加载上下文切换内容
}

//初始化
func New() *Engine {
	engine := &Engine{
		routerGroup: routerGroup{},
	}
	engine.pool.New = func() any {
		return engine.allocateContext() //后续可能增加很多的属性
	}
	return engine
}
func (e *Engine) allocateContext() any {
	return &Context{engine: e}
}

/*
···························································渲染功能模块························································
*/

//渲染html的三个接口
func (e *Engine) SetFuncMap(funcMap template.FuncMap) {
	e.funcMap = funcMap
}

func (e *Engine) LoadTemplate(pattern string) {
	t := template.Must(template.New("").Funcs(e.funcMap).ParseGlob(pattern))
	e.SetHtmlTemplate(t)
}
func (e *Engine) SetHtmlTemplate(t *template.Template) {
	e.HTMLRender = render.HTMLRender{Template: t}
}

/*
·······················································启动服务引擎························································
*/

func (e *Engine) httpRequestHandle(ctx *Context, w http.ResponseWriter, r *http.Request) {
	method := r.Method
	//对不同的请求做不同的处理
	for _, g := range e.groups {
		//遍历每一个路由
		//不能使用r.RequestURI需要使用
		routerName := SubstringLast(r.URL.Path, "/"+g.groupName)
		node := g.treeNode.Get(routerName)
		if node != nil && node.isEnd {
			//如果路由匹配上了
			handler, ok := g.handlerMap[node.routerName][ANY]
			if ok {
				g.methodHandle(node.routerName, ANY, handler, ctx)
				return
			}
			//对不同method进行匹配
			handler, ok = g.handlerMap[node.routerName][method]
			if ok {
				g.methodHandle(node.routerName, method, handler, ctx)
				return
			}
			//如果不匹配 405状态
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, r.RequestURI+""+method+"NOT ALLOWED")
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, r.RequestURI+""+method+"NOT FOUND")
}

//实现serverhttp 则说明也可以作为一个handler
func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := e.pool.Get().(*Context)
	ctx.W = w
	ctx.R = r
	e.httpRequestHandle(ctx, w, r)

	e.pool.Put(ctx)
}
func (e *Engine) Run() {
	http.Handle("/", e)
	err := http.ListenAndServe(":8111", nil)
	if err != nil {
		log.Fatal(err)
	}
}
