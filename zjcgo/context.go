package zjcgo

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/zhengjingcheng/zjcgo/binding"
	"github.com/zhengjingcheng/zjcgo/render"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
)

/*
	这个文件存放的是上下文信息
*/
//http.ResponseWriter接口是处理器用来构造HTTP响应的接口，包含三个方法签名。
//Header()	用户设置或获取响应头信息
//Write()	用于写入数据到响应体
//WriteHeader()	用于设置响应状态码，若不调用则默认状态码为200 OK。

//http.Request 服务器请求

const (
	defaultMaxMemory       = 32 << 20
	defaultMultipartMemory = 32 << 20
)

type Context struct {
	W                     http.ResponseWriter
	R                     *http.Request
	engine                *Engine
	queryCache            url.Values //提取get url参数
	fromCache             url.Values //提取 post url参数
	DisallowUnknownFields bool       //是否开启参数校验功能
	IsValidate            bool       //是否开启结构体检验功能(参数严格匹配)
}

/*
·············································参数提取模块（提取正常参数）·····················································
*/

//http://xxx.com/user/add?id=1&age=20&username=张三
//初始化
func (c *Context) initQueryCache() {
	if c.R != nil {
		c.queryCache = c.R.URL.Query()
	} else {
		c.queryCache = url.Values{}
	}
}

//返回带默认值的
func (c *Context) GetDefaultQuery(key, defaultValue string) string {
	array, ok := c.GetQueryArray(key)
	if !ok {
		return defaultValue
	}
	return array[0]
}

func (c *Context) GetQuery(key string) string {
	c.initQueryCache()
	return c.queryCache.Get(key)
}

//一个Key
func (c *Context) QueryArray(key string) (values []string) {
	c.initQueryCache()
	values, _ = c.queryCache[key]
	return
}

//一个key对应多个value
func (c *Context) GetQueryArray(key string) (values []string, ok bool) {
	c.initQueryCache()
	values, ok = c.queryCache[key]
	return
}

/*
·············································参数提取模块（提取map参数）·····················································
*/

//获得map参数:http://localhost:8080/queryMap?user[id]=1&user[name]=张三
func (c *Context) QueryMap(key string) (dicts map[string]string) {
	dicts, _ = c.GetQueryMap(key)
	return
}
func (c *Context) GetQueryMap(key string) (map[string]string, bool) {
	c.initQueryCache()
	return c.get(c.queryCache, key)
}
func (c *Context) get(cache map[string][]string, key string) (map[string]string, bool) {
	//user[id]=1&user[name]=张三
	dicts := make(map[string]string)
	exist := false
	for k, value := range cache {
		//先判断前边这部分有没有
		if i := strings.IndexByte(k, '['); i >= 1 && k[0:i] == key {
			//判断后边这部分有没有
			if j := strings.IndexByte(k[i+1:], ']'); j >= 1 {
				exist = true

				dicts[k[i+1:][:j]] = value[0]
			}
		}
	}
	return dicts, exist
}

/*
·············································参数提取模块（提取post参数）·····················································
*/
func (c *Context) initFormCache() {
	if c.fromCache == nil {
		c.fromCache = make(url.Values) //如果没有表单参数缓存就创建出来
		req := c.R
		//如果有错误的话打印出来
		if err := req.ParseMultipartForm(defaultMaxMemory); err != nil {
			if !errors.Is(err, http.ErrNotMultipart) {
				log.Println(err)
			}
		}
		//拿出表单缓存
		c.fromCache = c.R.PostForm
	}
}

//得到表单参数
func (c *Context) GetPostFormArray(key string) (values []string, ok bool) {
	c.initFormCache()
	values, ok = c.fromCache[key]
	return
}

func (c *Context) PostFormArray(key string) (values []string) {
	values, _ = c.GetPostFormArray(key)
	return
}

//返回获得的表单参数
func (c *Context) GetPostForm(key string) (string, bool) {
	if values, ok := c.GetPostFormArray(key); ok {
		return values[0], ok
	}
	return "", false
}
func (c *Context) GetPostFormMap(key string) (map[string]string, bool) {
	c.initFormCache()
	return c.get(c.fromCache, key)
}
func (c *Context) PostFormMap(key string) (dicts map[string]string) {
	dicts, _ = c.GetPostFormMap(key)
	return
}

/*
·············································参数提取模块（提取文件）·····················································
*/
//提供单个文件
func (c *Context) FormFile(name string) (*multipart.FileHeader, error) {
	req := c.R
	if err := req.ParseMultipartForm(defaultMultipartMemory); err != nil {
		return nil, err
	}
	file, header, err := req.FormFile(name)
	if err != nil {
		return nil, err
	}
	err = file.Close()
	if err != nil {
		return nil, err
	}
	return header, nil
}

//提供多个文件
func (c *Context) MultipartForm() (*multipart.Form, error) {
	err := c.R.ParseMultipartForm(defaultMultipartMemory)
	return c.R.MultipartForm, err
}

//提供多个文件2
func (c *Context) FormFiles(name string) []*multipart.FileHeader {
	multipartForm, err := c.MultipartForm()
	if err != nil {
		return make([]*multipart.FileHeader, 0)
	}
	return multipartForm.File[name]
}

/*
·············································参数提取模块（支持json参数）·····················································
*/
//解析json参数
func (c *Context) DealJson(data any) error {
	body := c.R.Body
	//post传参的内容是放在body中的
	if c.R == nil || body == nil {
		return errors.New("invalid request")
	}
	//自带的json解码器
	decoder := json.NewDecoder(body)
	//是否开启参数校验功能
	if c.DisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	return decoder.Decode(data)
}

/*
·············································参数提取模块（添加结构体校验功能）·····················································
*/

func validateRequireParam(data any, decoder *json.Decoder) error {
	//解析为map,然后根据Map中的key进行对比
	//判断类型 结构体 才能解析为Map
	//要用到reflect
	if data == nil {
		return nil
	}
	valueOf := reflect.ValueOf(data)
	//判断其是不是指针类型
	if valueOf.Kind() != reflect.Pointer {
		return errors.New("no ptr type")
	}
	//拿到元素类型
	t := valueOf.Elem().Interface()
	of := reflect.ValueOf(t)
	switch of.Kind() {
	case reflect.Struct: //如果是结构体就解析为Map
		return checkParam(of, data, decoder)
		//将Madata转成json
	case reflect.Slice, reflect.Array: //如果是切片或者数组
		elem := of.Type().Elem()
		if elem.Kind() == reflect.Struct {
			return checkParamSlice(elem, data, decoder)
		}
	default:
		err := decoder.Decode(data)
		if err != nil {
			return err
		}
	}
	return nil
}

func checkParam(of reflect.Value, data any, decoder *json.Decoder) error {
	mapData := make(map[string]interface{})
	_ = decoder.Decode(&mapData)
	for i := 0; i < of.NumField(); i++ {
		field := of.Type().Field(i)
		tag := field.Tag.Get("json")
		required := field.Tag.Get("zjcgo")
		value := mapData[tag]
		//判断有没有这个json tag 并且这个字段还是个必须字段
		if value == nil && required == "required" {
			return errors.New(fmt.Sprintf("filed [%s] is not exist", tag))
		}
	}
	marshal, _ := json.Marshal(mapData)
	_ = json.Unmarshal(marshal, data)
	return nil
}
func checkParamSlice(elem reflect.Type, data any, decoder *json.Decoder) error {
	mapData := make([]map[string]interface{}, 0)
	_ = decoder.Decode(&mapData)
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		required := field.Tag.Get("zjcgo")
		//name := field.Name
		jsonName := field.Tag.Get("json")
		for _, v := range mapData {
			value := v[jsonName]
			if value == nil && required == "required" {
				return errors.New(fmt.Sprintf("filed [%s] is required", jsonName))
			}
		}
	}
	marshal, _ := json.Marshal(mapData)
	_ = json.Unmarshal(marshal, data)
	return nil
}
func (c *Context) DealJsonnew(data any) error {
	body := c.R.Body
	if c.R == nil || body == nil {
		return errors.New("invalid request")
	}
	decoder := json.NewDecoder(body)
	//是否开启参数校验功能
	if c.DisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	if c.IsValidate {
		//结构体校验
		err := validateRequireParam(data, decoder)
		if err != nil {
			return err
		}
	} else {
		err := decoder.Decode(data)
		if err != nil {
			return err
		}
	}
	return nil
}

/*
····················································参数提取模块（第三方校验功能）·······················································
*/
func (c *Context) BindJson(obj any) error {
	jsonBinding := binding.JSON
	return c.MustBindWith(obj, jsonBinding)
}

func (c *Context) MustBindWith(obj any, b binding.Binding) error {
	//如果发生错误，返回400状态码 参数错误
	if err := c.ShouldBindWith(obj, b); err != nil {
		c.W.WriteHeader(http.StatusBadRequest)
		return err
	}
	return nil
}

func (c *Context) ShouldBindWith(obj any, b binding.Binding) error {
	return b.Bind(c.R, obj)
}

/*
····················································参数提取模块（xml参数提取）·······················································
*/
func (c *Context) BindXML(obj any) error {
	return c.MustBindWith(obj, binding.XML)
}

/*
·····················································页面渲染模块·························································
*/

//最简单的HTML函数
func (c *Context) HTML(status int, html string) error {
	//状态是200 默认不设置的话 如果调用了write这个方法 实际上返回状态 200
	return c.Render(status, &render.HTML{Data: html, IsTemplate: false})
}

//实现提前加入模板的页面渲染函数
func (c *Context) Template(name string, data any) error {
	return c.Render(http.StatusOK, &render.HTML{Data: data, IsTemplate: true, Template: c.engine.HTMLRender.Template, Name: name})
}

//支持渲染jason格式
func (c *Context) JSON(status int, data any) error {
	return c.Render(status, &render.JSON{
		Data: data,
	})
}

//支持渲染jason格式
func (c *Context) XML(status int, data any) error {

	return c.Render(status, &render.XML{
		Data: data,
	})
}

//支持下载文件
func (c *Context) File(fileName string) {
	http.ServeFile(c.W, c.R, fileName)
}
func (c *Context) FileAttachment(filepath, filename string) {
	if isASCII(filename) {
		c.W.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	} else {
		c.W.Header().Set("Content-Disposition", `attachment; filename*=UTF-8''`+url.QueryEscape(filename))
	}
	http.ServeFile(c.W, c.R, filepath)
}
func (c *Context) FileFromFS(filepath string, fs http.FileSystem) {
	defer func(old string) {
		c.R.URL.Path = old
	}(c.R.URL.Path)

	c.R.URL.Path = filepath

	http.FileServer(fs).ServeHTTP(c.W, c.R)
}

//支持重定向
func (c *Context) Redirect(status int, url string) error {
	return c.Render(status, &render.Redirect{Code: status, Request: c.R, Location: url})
}

//支持string
func (c *Context) String(status int, format string, values ...any) error {
	return c.Render(status, &render.String{Format: format, Data: values})
}

func (c *Context) Render(statusCode int, r render.Render) error {
	err := r.Render(c.W)
	if statusCode != http.StatusOK {
		c.W.WriteHeader(statusCode)
	}
	return err
}

/*
·····················································保存文件模块·························································
*/
func (c *Context) SaveUploadedFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}
