package zjcpool

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type sig struct {
}

const DefaultExpire = 2

var (
	ErrorInValidCap    = errors.New("pool cap can not <= 0")
	ErrorInValidExpire = errors.New("pool expire cap can not <= 0")
	ErrorHasClosed     = errors.New("pool has bean released!!") //资源被释放
)

type Pool struct {
	//容量 pool max 容量
	cap int32
	//running 正在运行的worker数量
	running int32
	//空闲worker
	workers []*Worker
	//过期时间 空闲的worker超过这个时间就回收掉 （单开线程不停循环，在新建线程池时就进行检测）
	expire time.Duration
	//释放资源 pool就不能使用了
	release chan sig
	//lock 保护Pool里边的相关资源的安全
	lock sync.Mutex
	//once 释放只能调用一次 不能多次调用
	once sync.Once
	//缓存优化
	workerCache sync.Pool
	//信号量优化
	cond *sync.Cond
	//
	PanicHandler func()
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
		cap: cap,
		//一个work最多可以持续多久
		expire:  time.Duration(expire) * time.Second,
		release: make(chan sig, 1),
	}
	p.workerCache.New = func() any {
		return &Worker{
			pool: p,
			task: make(chan func(), 1),
		}
	}
	p.cond = sync.NewCond(&p.lock)
	//另外开一个线程定时清理过期的空闲worker
	go p.expireWorker()
	return p, nil
}

func (p *Pool) expireWorker() {
	//定时清理过期的空闲worker
	ticker := time.NewTicker(p.expire)
	for range ticker.C {
		if p.IsClosed() {
			break
		}
		//循环空闲的workers 如果当前时间和worker的最后运行任务的时间 差值大于expire 进行清理
		p.lock.Lock()
		idleWorkers := p.workers
		n := len(idleWorkers) - 1
		if n >= 0 {
			var clearN = -1
			for i, w := range idleWorkers {
				if time.Now().Sub(w.lastTime) <= p.expire {
					break
				}
				clearN = i
				w.task <- nil
				idleWorkers[i] = nil
			}
			// 3 2
			if clearN != -1 {
				if clearN >= len(idleWorkers)-1 {
					p.workers = idleWorkers[:0]
				} else {
					// len=3 0,1 del 2
					p.workers = idleWorkers[clearN+1:]
				}
				fmt.Printf("清除完成,running:%d, workers:%v \n", p.running, p.workers)
			}
		}
		p.lock.Unlock()
	}
}

//提交任务
func (p *Pool) Submit(task func()) error {
	if len(p.release) > 0 {
		//如果已经被释放了
		return ErrorHasClosed
	}
	//获取池里面得worker，然后执行任务就可以了
	w := p.GetWorker()
	w.task <- task //把任务通过管道传给工人
	w.run()
	return nil
}

func (p *Pool) GetWorker() *Worker {
	//如果有空闲的worker 直接获取
	p.lock.Lock()
	idleWokers := p.workers
	n := len(idleWokers) - 1
	if n >= 0 {
		w := idleWokers[n]
		idleWokers[n] = nil
		p.workers = idleWokers[:n]
		p.lock.Unlock()
		return w
	}
	//如果没有空闲的worker 要新建一个worker
	if p.running < p.cap {
		//还不够pool的容量,直接新建一个
		p.lock.Unlock()
		c := p.workerCache.Get()
		var w *Worker
		if c == nil {
			w = &Worker{
				pool: p,
				task: make(chan func(), 1),
			}
		} else {
			w = c.(*Worker)
		}
		return w
	}
	p.lock.Unlock()
	//如果正在运行的wokers大于pool 容量阻塞等待，worker释放
	//用条件变量优化等待释放
	return p.waitIdleWorker()
}

func (p *Pool) waitIdleWorker() *Worker {
	p.lock.Lock()
	p.cond.Wait()
	idleWokers := p.workers
	n := len(idleWokers) - 1
	if n < 0 {
		p.lock.Unlock()
		if p.running < p.cap {
			//还不够pool的容量,直接新建一个
			c := p.workerCache.Get()
			var w *Worker
			if c == nil {
				w = &Worker{
					pool: p,
					task: make(chan func(), 1),
				}
			} else {
				w = c.(*Worker)
			}
			return w
		}
		return p.waitIdleWorker()
	}
	w := idleWokers[n]
	idleWokers[n] = nil
	p.workers = idleWokers[:n]
	p.lock.Unlock()
	return w
}

func (p *Pool) incRunnig() {
	atomic.AddInt32(&p.running, 1)
}

func (p *Pool) PutWorker(w *Worker) {
	//更新最后使用时间
	w.lastTime = time.Now()
	p.lock.Lock()
	p.workers = append(p.workers, w)
	p.cond.Signal() //通知有空闲
	p.lock.Unlock()
}

func (p *Pool) decRunning() {
	atomic.AddInt32(&p.running, -1)
}

//释放资源
func (p *Pool) Release() {
	p.once.Do(func() {
		//只执行一次
		p.lock.Lock()
		workers := p.workers
		for i, w := range workers {
			w.task = nil
			w.pool = nil
			workers[i] = nil
		}
		p.workers = nil
		p.lock.Unlock()
		p.release <- sig{}
	})
}

//判断是不是已经关闭了
func (p *Pool) IsClosed() bool {
	return len(p.release) > 0
}
func (p *Pool) Restart() bool {
	if len(p.release) <= 0 {
		return true
	}
	_ = <-p.release
	//重启之后要开启清除功能
	go p.expireWorker()
	return true
}
