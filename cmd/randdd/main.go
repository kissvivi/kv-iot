package main

import (
	"fmt"
	"math/rand"
	"time"
)

// 生成不重复随机数切片
func generateRandomNumbers(min, max int, count int) []int {
	if max-min+1 < count {
		return nil
	}

	// 初始化整数切片并填充从min到max的连续整数
	numbers := make([]int, max-min+1)
	for i := range numbers {
		numbers[i] = min + i
	}

	// 设置随机数种子，一般使用当前时间作为种子
	rand.Seed(time.Now().UnixNano())

	// 使用洗牌算法打乱整数切片
	for i := len(numbers) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		numbers[i], numbers[j] = numbers[j], numbers[i]
	}

	// 返回不重复的随机数切片
	return numbers[:count]
}

func main() {

	for i := 0; i < 4; i++ {

		// 定义要生成随机数的区间和数量
		min := 0
		max := 1
		count := 2

		// 生成不重复的随机数切片
		randomNumbers := generateRandomNumbers(min, max, count)

		if randomNumbers == nil {
			fmt.Println("无法生成不重复的随机数切片，数量超过区间范围。")
			return
		}

		// 输出生成的不重复随机数
		fmt.Println("生成的不重复随机数：", randomNumbers)

		time.Sleep(50)
	}
}
