package zjcpool

import "time"

type Worker struct {
	pool *Pool
	//任务
	task chan func()
	//执行任务最后的时间
	lastTime time.Time
}

func (w *Worker) run() {
	go w.running()
}

func (w *Worker) running() {
	for f := range w.task {
		if f == nil {
			return
		}
		f()
		//任务运行完成,worker 空闲
		w.pool.PutWorker(w)
		//运行完之后减一
		w.pool.decRunning()
	}
}
