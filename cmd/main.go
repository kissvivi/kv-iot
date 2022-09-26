package main

import (
	"fmt"
	"math/rand"
	"time"
)

var _randNow int

func main() {

	for {
		nn := getRandomNext(_randNow, 2)
		_randNow = nn

		fmt.Println(nn, _randNow)
		time.Sleep(500)
	}

	fmt.Println("this is kv-iot")
}

func getRandomNext(index int, lens int) int {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var nowIndex = 0
	nowIndex = r.Intn(lens)
	var i int
	for {

		i++
		if nowIndex == index {
			nowIndex = r.Intn(lens)
		} else {
			index = nowIndex
			break
		}
	}
	fmt.Println("寻找次数：", i)
	return index
}
