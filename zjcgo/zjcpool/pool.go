package zjcpool

import (
	"errors"
	"sync"
	"time"
)

type sig struct {
}

const DefaultExpire = 3

var (
	ErrorInValidCap    = errors.New("pool cap can not <= 0")
	ErrorInValidExpire = errors.New("pool expire cap can not <= 0")
)

type Pool struct {
	//容量 pool max 容量
	cap int
	//running 正在运行的worker数量
	//空闲worker
	workers []*Worker
	//过期时间 空闲的worker超过这个时间就回收掉
	expire time.Duration
	//释放资源 pool就不能使用了
	release chan sig
	//lock 保护Pool里边的相关资源的安全
	lock sync.Mutex
	//once 释放只能调用一次 不能多次调用
	once sync.Once
}

func NewPool(cap int) (*Pool, error) {
	return NewTimePool(cap, DefaultExpire)
}
func NewTimePool(cap int, expire int) (*Pool, error) {
	if cap <= 0 {
		return nil, ErrorInValidCap
	}
	if expire <= 0 {
		return nil, ErrorInValidExpire
	}
	p := &Pool{
		cap:     int(cap),
		expire:  time.Duration(expire) * time.Second,
		release: make(chan sig, 1),
	}
	return p, nil
}

//提交任务
func (p *Pool) Submit(task func()) error {
	//获取池里面得worker，然后执行任务就可以了
	w := p.GetWorker()
	w.task <- task
	return nil
}

func (p *Pool) GetWorker() *Worker {
	return nil
}
