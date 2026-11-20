package main

import (
	"fmt"
	"time"
)

// 平方取中法随机数生成器结构体
type SquareRootMidpoint struct {
	seed int64 // 种子
}

// 创建新的随机数生成器实例
func NewSquareRootMidpoint() *SquareRootMidpoint {
	return &SquareRootMidpoint{
		seed: time.Now().UnixNano(),
	}
}

// 生成下一个随机数
func (s *SquareRootMidpoint) Next() int64 {
	s.seed = s.seed * s.seed // 平方
	sStr := fmt.Sprintf("%d", s.seed)

	// 取中间的几位数作为随机数
	start := len(sStr) / 4
	end := len(sStr) - start
	sStr = sStr[start:end]

	// 转换成整数返回
	var result int64
	fmt.Sscanf(sStr, "%d", &result)

	return result
}

func main() {
	rng := NewSquareRootMidpoint()

	// 生成 10 个随机数并输出
	fmt.Println("Generated random numbers:")
	for i := 0; i < 10; i++ {
		fmt.Println(rng.Next())
	}
}
