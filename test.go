package main

import "fmt"

//模拟组装2台电脑，
//--- 抽象层 ---有显卡Card  方法display，有内存Memory 方法storage，有处理器CPU 方法calculate
//--- 实现层层 ---有 Intel因特尔公司 、产品有(显卡、内存、CPU)，有 Kingston 公司， 产品有(内存3)，有 NVIDIA 公司， 产品有(显卡)
//--- 逻辑层 ---1. 组装一台Intel系列的电脑，并运行，2. 组装一台 Intel CPU  Kingston内存 NVIDIA显卡的电脑，并运行

type Card interface {
	display()
}
type Memory interface {
	storage()
}
type CPU interface {
	calculate()
}
type com struct {
	Card
	Memory
	CPU
}

func Newcmp(card Card, memory Memory, cpu CPU) *com {
	return &com{Card: card, Memory: memory, CPU: cpu}
}
func (cmp *com) work() {
	cmp.display()
	cmp.storage()
	cmp.calculate()
}

type InterCpu struct {
	CPU
}

type InterCard struct {
	Card
}

type InterMemory struct {
	Memory
}

type NVIDIACard struct {
	Card
}

type NVIDIAMemory struct {
	Memory
}
type NVIDIACpu struct {
	CPU
}

func (*InterCard) display() {
	fmt.Println("因特尔显卡开始显示")
}
func (*InterMemory) storage() {
	fmt.Println("因特尔内存开始存储")
}
func (*InterCpu) calculate() {
	fmt.Println("因特尔CPU开始运行")
}

func (*NVIDIACard) display() {
	fmt.Println("英伟达显卡开始显示")
}
func (*NVIDIAMemory) storage() {
	fmt.Println("英伟达内存开始存储")
}
func (*NVIDIACpu) calculate() {
	fmt.Println("英伟达CPU开始运行")
}
func main() {
	a := Newcmp(&NVIDIACard{}, &InterMemory{}, &NVIDIACpu{})
	a.work()
}
