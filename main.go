package main

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

var wg sync.WaitGroup

func producer(out chan<- int) {
	defer wg.Done()
	for i := 1; i <= 100; i++ {
		out <- i // 缓冲区写入数据
		if i%2 != 0 {
			fmt.Println("生产者生产数据:", i)
		}
	}
	close(out) //写完关闭管道
}

func consumer(in <-chan int) {
	defer wg.Done()
	// 无需同步机制，先做后做
	// 没有数据就阻塞等
	for i := 1; i <= 100; i++ {
		<-in
		if i%2 == 0 {
			fmt.Println("消费者读取数据:", i)
		}
	}

}

type People struct {
	Name string
	Id   int64
}

func main() {
	// 传参的时候显式类型像隐式类型转换，双向管道向单向管道转换
	var str strings.Builder
	var s []string
	s = append(s, "a")
	s = append(s, "a")
	s = append(s, "A")
	join := strings.Join(s, "V")
	upper := strings.ToUpper(join)
	fmt.Println(upper)
	str.WriteString("zjc")
	fmt.Println(str.String())
	people := &People{
		Name: "zjc1",
		Id:   123,
	}
	b := reflect.TypeOf(people)
	a := reflect.ValueOf(people)
	for i := 0; i < b.Elem().NumField(); i++ {
		fmt.Println(b.Elem().Field(i).Name)
		fmt.Println(a.Elem().Field(i).Interface())
	}
	//time.Sleep(time.Second * 10)
}
