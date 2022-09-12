package service

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/zhengjingcheng/zjcgo/orm"
	"net/url"
)

type User struct {
	Id       int64 `zjcorm:"id,auto_increment"`
	Username string
	Password string
	Age      int
}

//插入单个
func SaveUser() {
	dataSourceName := fmt.Sprintf("root:zjc19980924@tcp(localhost:3306)/zjcgo?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
	db := orm.Open("mysql", dataSourceName)
	//可以给前缀
	//db.Prefix = "zjcgo_"
	user := &User{
		Id:       10001,
		Username: "zjc",
		Password: "112321",
		Age:      20,
	}
	id, _, err := db.New(&User{}).Insert(user)
	if err != nil {
		panic(err)
	}
	fmt.Println(id)
	db.Close()
}

//插入批量
func SaveUserBatch() {
	dataSourceName := fmt.Sprintf("root:zjc19980924@tcp(localhost:3306)/zjcgo?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
	db := orm.Open("mysql", dataSourceName)
	//可以给前缀
	//db.Prefix = "zjcgo_"
	user := &User{
		Username: "zjc",
		Password: "112321",
		Age:      20,
	}
	user1 := &User{
		Username: "zjc",
		Password: "112321",
		Age:      23,
	}
	var users []any
	users = append(users, user1, user)
	id, _, err := db.New(&User{}).InsertBash(users)
	if err != nil {
		panic(err)
	}
	fmt.Println(id)
	db.Close()
}
func UpdateUser() {
	dataSourceName := fmt.Sprintf("root:zjc19980924@tcp(localhost:3306)/zjcgo?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
	db := orm.Open("mysql", dataSourceName)
	//可以给前缀
	//db.Prefix = "zjcgo_"
	user := &User{
		Username: "zjc",
		Password: "5243",
		Age:      23,
	}
	_, _, err := db.New(&User{}).Where("id", 1006).Update(user)
	if err != nil {
		panic(err)
	}

	db.Close()
}

func SelectOne() {
	dataSourceName := fmt.Sprintf("root:zjc19980924@tcp(localhost:3306)/zjcgo?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
	db := orm.Open("mysql", dataSourceName)
	//可以给前缀
	//db.Prefix = "zjcgo_"
	user := &User{}
	err := db.New(user).Where("id", 1006).SelectOne(user, "id")
	if err != nil {
		panic(err)
	}
	fmt.Println(user)
	db.Close()
}
func Select() {
	dataSourceName := fmt.Sprintf("root:zjc19980924@tcp(localhost:3306)/zjcgo?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
	db := orm.Open("mysql", dataSourceName)
	//可以给前缀
	//db.Prefix = "zjcgo_"
	user := &User{}
	users, err := db.New(user).Where("id", 1000).Order("id", "asc").Select(user)
	if err != nil {
		panic(err)
	}
	for _, v := range users {
		u := v.(*User)
		fmt.Println(u)
	}
	fmt.Println(users)
	db.Close()
}
func Count() {
	dataSourceName := fmt.Sprintf("root:zjc19980924@tcp(localhost:3306)/zjcgo?charset=utf8&loc=%s&parseTime=true", url.QueryEscape("Asia/Shanghai"))
	db := orm.Open("mysql", dataSourceName)
	//可以给前缀
	//db.Prefix = "zjcgo_"
	user := &User{}
	count, err := db.New(user).Count()
	if err != nil {
		panic(err)
	}
	fmt.Println(count)
	db.Close()
}
