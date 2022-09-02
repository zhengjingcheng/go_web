package zjcpool

import (
	zjcLog "github.com/zhengjingcheng/zjcgo/log"
	"time"
)

type Worker struct {
	pool *Pool
	//任务
	task chan func()
	//执行任务最后的时间
	lastTime time.Time
}

func (w *Worker) run() {
	w.pool.incRunnig()
	go w.running()
}

func (w *Worker) running() {
	defer func() {
		//任务运行完成,worker 空闲
		w.pool.decRunning()
		w.pool.workerCache.Put(w)
		if err := recover(); err != nil {
			//捕获任务发生的panic
			if w.pool.PanicHandler != nil {
				w.pool.PanicHandler()
			} else {
				zjcLog.Default().Error(err)
			}
		}
		w.pool.cond.Signal()
	}()
	for f := range w.task {
		if f == nil {
			//用完了就放回去
			w.pool.workerCache.Put(w)
			return
		}
		f()
		//任务运行完成,worker 空闲
		w.pool.PutWorker(w)
	}
}
