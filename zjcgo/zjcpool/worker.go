package zjcpool

import "time"

type Worker struct {
	pool *Pool
	//任务
	task chan func()
	//执行任务最后的时间
	lastTime time.Time
}
