package main

import (
	"fmt"
	"github.com/zhengjingcheng/zjcgo"
	"log"
	"net/http"
)

type User struct {
	Name      string   `xml:"name" json:"name" validate:"required"`
	Age       int      `xml:"age" json:"age" validate:"required,max=50,min=10"`
	Addresses []string `json:"addresses"`
	//	Email     string   `json:"email" validate:"required"`
}

func Log(next zjcgo.HandlerFunc) zjcgo.HandlerFunc {
	return func(ctx *zjcgo.Context) {
		fmt.Println("打印请求参数")
		next(ctx)
		fmt.Println("返回执行时间")
	}
}
func main() {
	engine := zjcgo.New()     //起一个服务引擎
	g := engine.Group("user") //将路由组的名字加进去
	g.Use(func(next zjcgo.HandlerFunc) zjcgo.HandlerFunc {
		return func(ctx *zjcgo.Context) {
			fmt.Println("pre handler")
			next(ctx)
			fmt.Println("POST handler")
		}
	})
	//路由处理函数
	g.Get("/hello", func(ctx *zjcgo.Context) {
		fmt.Println("handler")
		fmt.Fprint(ctx.W, "欢迎来到郑金成的博客")
	}, Log)
	g.Post("/info", func(ctx *zjcgo.Context) {
		fmt.Fprint(ctx.W, "pos欢迎来到郑金成的博客")
	})
	g.Any("/any", func(ctx *zjcgo.Context) {
		fmt.Fprint(ctx.W, "any欢迎来到郑金成的博客")
	})
	g.Get("/get/:id", func(ctx *zjcgo.Context) {
		fmt.Fprintf(ctx.W, "%s /get/*/set user info path variable", "zjccom")
	})

	g.Get("/html", func(ctx *zjcgo.Context) {
		ctx.HTML(http.StatusOK, "<h1>zjc博客</h1>")
	})

	engine.LoadTemplate("tpl/*.html")

	g.Get("/template", func(ctx *zjcgo.Context) {
		user := &User{
			Name: "ZJC",
		}
		err := ctx.Template("login.html", user)
		if err != nil {
			log.Panic(err)
		}
	})

	g.Get("/json", func(ctx *zjcgo.Context) {
		user := &User{
			Name: "ZJC",
		}
		err := ctx.JSON(http.StatusOK, user)
		if err != nil {
			log.Panic(err)
		}
	})
	g.Get("/xml", func(ctx *zjcgo.Context) {
		user := &User{
			Name: "ZJC",
			Age:  10,
		}
		err := ctx.XML(http.StatusOK, user)
		if err != nil {
			log.Panic(err)
		}
	})
	g.Get("/excel", func(ctx *zjcgo.Context) {
		ctx.File("tpl/text.xlsx")
	})
	g.Get("/excelName", func(ctx *zjcgo.Context) {
		ctx.FileAttachment("tpl/text.xlsx", "aaaa")
	})
	g.Get("/fs", func(ctx *zjcgo.Context) {
		ctx.FileFromFS("text.xlsx", http.Dir("tpl"))
	})
	g.Get("/redirect", func(ctx *zjcgo.Context) {
		ctx.Redirect(http.StatusFound, "/user/hello")
	})
	//参数模块测试
	//测试提取任意普通参数
	g.Get("/add", func(ctx *zjcgo.Context) {
		name := ctx.GetDefaultQuery("name", "张三")
		fmt.Printf("name:%v,ok:%v", name, true)
	})
	//测试提取map参数
	g.Get("/queryMap", func(ctx *zjcgo.Context) {
		m, _ := ctx.GetQueryMap("user")
		ctx.JSON(http.StatusOK, m)
	})

	//测试提交表单参数
	g.Post("/add", func(ctx *zjcgo.Context) {
		name, _ := ctx.GetPostFormArray("name")
		fmt.Println(name)
	})
	//测试提交map参数
	g.Post("/add1", func(ctx *zjcgo.Context) {
		name, _ := ctx.GetPostFormMap("user")
		ctx.JSON(http.StatusOK, name)
	})
	//测试提取文件
	g.Post("/add2", func(ctx *zjcgo.Context) {
		name, _ := ctx.GetPostFormMap("user")
		ctx.JSON(http.StatusOK, name)
		//file, _ := ctx.FormFile("file")
		//err := ctx.SaveUploadedFile(file, "./upload/"+file.Filename)
		//if err != nil {
		//	log.Println(err)
		//}
		//同一个名字提取多个文件
		//form, err := ctx.MultipartForm()
		//log.Println(err)
		//fileMap := form.File
		//headers := fileMap["file"]
		//for _, file := range headers {
		//	ctx.SaveUploadedFile(file, "./upload/"+file.Filename)
		//}
		//同一个名字提取多个文件
		files := ctx.FormFiles("file")
		for _, file := range files {
			ctx.SaveUploadedFile(file, "./upload/"+file.Filename)
		}
	})

	//json
	g.Post("/jsonParm", func(ctx *zjcgo.Context) {
		user := make([]User, 0)
		//开启参数检验
		ctx.DisallowUnknownFields = true
		//开启结构体验证
		ctx.IsValidate = true
		err := ctx.BindJson(&user)
		if err == nil {
			ctx.JSON(http.StatusOK, user)
		} else {
			log.Println(err)
		}
	})

	//xml
	g.Post("/xmlParam", func(ctx *zjcgo.Context) {
		user := &User{}
		//user := User{}
		err := ctx.BindXML(user)
		if err == nil {
			ctx.JSON(http.StatusOK, user)
		} else {
			log.Println(err)
		}
	})
	engine.Run()
}
