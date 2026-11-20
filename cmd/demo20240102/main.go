package main

import (
	"fmt"
	"sync"
	"time"
)

type printDemo struct {
	ch    chan int        //传递打印数字
	count int             //计数
	ok    bool            //控制交替
	wg    *sync.WaitGroup //控制协程结束
}

func (p *printDemo) printNumber() {
	i := 1 //从1开始打印
	for {
		if p.ok {
			fmt.Println("print number is : ", i)
			p.ch <- i //传递给其他携程
			p.ok = false
			i++
		}
		if i > p.count {
			p.wg.Done()
			return
		}

	}
}

func (p *printDemo) printABC() {
	time.Sleep(10 * time.Microsecond)
	for {
		ch, ok := <-p.ch
		if ok && !p.ok {
			//通过ASCII码转换成abc
			fmt.Printf("print abc is : %s\n", string(rune(ch+96)))
			p.ok = true
		}
		if ch == p.count {
			fmt.Println("over")
			p.wg.Done()
			return
		}
	}
}

func main() {
	p := printDemo{}
	p.wg = &sync.WaitGroup{}
	p.wg.Add(2)
	p.count = 26
	p.ch = make(chan int, p.count)
	p.ok = true
	go p.printNumber()
	go p.printABC()
	p.wg.Wait()
}
