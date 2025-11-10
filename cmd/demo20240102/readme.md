
### golang中两个协程交替打印数字跟字母
### 知识点

1. 协程以及chan通道
2.  WaitGroup
3. ASCII码转换

#### 协程以及chan通道介绍
协程是Go的轻量级线程，可以让程序在并发执行多个任务；
通道是Go的一种数据结构，是协程之间传递数据的一种方式。
#### WaitGroup介绍
WaitGroup是协程并发控制的方法
WaitGroup 是内部通过一个计数器来统计有多少协程被等待。
这个计数器的值在我们启动 协程 之前先写入（使用 Add 方法）；
然后在 协程结束的时候，将这个计数器减 1（使用 Done 方法）；
启动协程后需要调用Wait 来进行等待，在 Wait 调用的地方会阻塞，直到 WaitGroup 内部的计数器减到 0。
#### ASCII码转换介绍

```go
// rune is an alias for int32 and is equivalent to
// int32 in all ways. It is
// used, by convention, to distinguish character
// values from integer values.

// int32的别名，几乎在所有方面等同于int32
// 它用来区分字符值和整数值

type rune = int32

// 我们通常会使用rune类型 来对 ASCII码与数字与字符之间的转换
// 详细转换对应关系 查ASCII码表
```

代码示例

```go
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
	p.wg.Add(1)
	p.count = 26
	p.ch = make(chan int, p.count)
	p.ok = true
	go p.printNumber()
	go p.printABC()
	p.wg.Wait()
}
```


