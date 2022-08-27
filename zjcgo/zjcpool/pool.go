package zjcpool

import (
	"errors"
	"sync"
	"sync/atomic"
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
	cap int32
	//running 正在运行的worker数量
	running int32
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

func NewPool(cap int32) (*Pool, error) {
	return NewTimePool(cap, DefaultExpire)
}
func NewTimePool(cap int32, expire int) (*Pool, error) {
	if cap <= 0 {
		return nil, ErrorInValidCap
	}
	if expire <= 0 {
		return nil, ErrorInValidExpire
	}
	p := &Pool{
		cap:     cap,
		expire:  time.Duration(expire) * time.Second,
		release: make(chan sig, 1),
	}
	return p, nil
}

//提交任务
func (p *Pool) Submit(task func()) error {
	//获取池里面得worker，然后执行任务就可以了
	w := p.GetWorker()
	w.task <- task //把任务通过管道传给工人
	w.pool.incRunnig()
	return nil
}

func (p *Pool) GetWorker() *Worker {
	//如果有空闲的worker 直接获取
	//	p.lock.Lock()
	idleWokers := p.workers
	n := len(idleWokers) - 1
	if n >= 0 {
		p.lock.Lock()
		w := idleWokers[n]
		idleWokers[n] = nil
		p.workers = idleWokers[:n]
		p.lock.Unlock()
		return w
	}
	//如果没有空闲的worker 要新建一个worker
	if p.running < p.cap {
		//还不够pool的容量,直接新建一个
		w := &Worker{
			pool: p,
			task: make(chan func(), 1),
		}
		w.run()
		return w
	}
	//如果正在运行的wokers+空闲的worker如果大于pool 容量阻塞等待，worker释放
	for {
		p.lock.Lock()
		idleWokers := p.workers
		n = len(idleWokers) - 1
		if n < 0 {
			p.lock.Unlock()
			continue
		}
		w := idleWokers[n]
		idleWokers[n] = nil
		p.workers = idleWokers[:n]
		p.lock.Unlock()
		return w
	}
}

func (p *Pool) incRunnig() {
	atomic.AddInt32(&p.running, 1)
}

func (p *Pool) PutWorker(w *Worker) {
	w.lastTime = time.Now()
	p.lock.Lock()
	p.workers = append(p.workers, w)
	p.lock.Unlock()
}

func (p *Pool) decRunning() {
	atomic.AddInt32(&p.running, -1)
}
