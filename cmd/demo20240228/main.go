package main

import "fmt"

//在一个平行宇宙中，麦当劳的麦乐鸡块分为7块装、13块装和29块装。有一天，你的老板让你出去购买正好为n块(0 < n <= 10000)的麦乐鸡块回来，
//请提供一个算法判断是否可行。
//例子：
//示例 1:
//输入: 20
//输出: true
//
//示例 2:
//输入: 19
//输出: false

func mljkk(n int) bool {
	nn := make([]bool, n)
	nn[0] = true
	for i := 1; i <= n; i++ {
		if i >= 7 && nn[i-7] {
			nn[i] = true
		} else if i >= 13 && nn[i-13] {
			nn[i] = true
		} else if i >= 29 && nn[i-29] {
			nn[i] = true
		}
	}
	return nn[n]
}
func main() {
	//b:=canBuyMcNuggets2(55)
	//
	//fmt.Print("n=20,输出：",b)
	//	merge([]int{1,2,3,0,0,0},3,[]int{1,2,3},3)
	//	e:=removeElement([]int{3,2,2,3},3)
	//	fmt.Println(e)
	fmt.Println(mljkk(29))
}

func mljk(x, y, z, n int) bool {
	//for  {

	v := 7*x + 13*y + 29*z
	if v > 0 && n%v == 0 {
		return true
	} else if n%7 > 7 && n > 0 {
		x = x + 1
	} else if n%13 > 7 && n > 0 {
		y = y + 1
	} else if n%29 > 7 && n > 0 {
		z = z + 1
	}
	//}
	mljk(x, y, z, n)
	return false
}

func canBuyMcNuggets(n int) bool {
	dp := make([]bool, n+1)
	dp[0] = true

	for i := 1; i <= n; i++ {
		if i >= 7 && dp[i-7] {
			fmt.Printf("i:%d,dp[i]:%v,n:%d\n", i, dp[i], n)
			dp[i] = true
		} else if i >= 13 && dp[i-13] {
			dp[i] = true
			fmt.Printf("i:%d,dp[i]:%v,n:%d\n", i, dp[i], n)
		} else if i >= 29 && dp[i-29] {
			dp[i] = true
			fmt.Printf("i:%d,dp[i]:%v,n:%d\n", i, dp[i], n)
		}
		fmt.Printf("i:%d,dp[i]:%v,n:%d,dp[n]:%v\n", i, dp[i], n, dp[n])
	}

	return dp[n]
}

func canBuyMcNuggets2(n int) bool {
	// 分别代表三种包装的数量

	// 尽可能多地使用最大的包装
	n %= 29

	// 尽可能多地使用次大的包装
	n %= 13

	// 使用最小的包装
	n %= 7

	// 如果能够将所有的包装都用完，并且剩余鸡块数为 0，则说明可以正好购买
	return n == 0
}

func merge(nums1 []int, m int, nums2 []int, n int) {

	for i := m; i < len(nums1); i++ {
		if m == 0 {
			nums1[0] = nums2[0]
		} else if m < n {
			nums1[i] = nums2[n-i]
		} else {
			nums1[i] = nums2[i-n]
		}

	}

	//for i := 0; i < len(nums1); i++ {
	//	index := i
	//	for j := i; j < len(nums1); j++ {
	//
	//		if nums1[j]<nums1[index]{
	//			index = j
	//		}
	//	}
	//	nums1[index], nums1[i] = nums1[i], nums1[index]
	//}
	nums1 = quickSort(nums1)

	fmt.Println(nums1)
}

func quickSort(arr []int) []int {
	if len(arr) <= 1 {
		return arr
	}
	p := arr[len(arr)/2]

	var left, right []int

	for _, num := range arr {

		if p > num {
			left = append(left, num)
		} else if p < num {
			right = append(right, num)
		}
	}

	left = quickSort(left)
	right = quickSort(right)

	return append(append(left, p), right...)

}

func removeElement(nums []int, val int) int {

	for {
		fmt.Println("111", nums)
		p := EerFen(nums, val)
		fmt.Println("元素位置：", p)
		if p == -1 {
			break
		} else {
			nums = append(nums[:p], nums[p+1:]...)
			fmt.Println(nums)
		}
	}
	return len(nums)
}

func EerFen(arr []int, target int) int {
	left, right := 0, len(arr)-1
	for left <= right {
		if left > right {
			return -1 // 如果左边索引大于右边索引，返回-1
		}
		p := left + (right-left)/2
		if arr[p] == target {
			return p
		} else if arr[p] < target {
			if arr[left] > arr[right] { // 左边大于右边时，调整搜索方向
				right = p - 1
			} else {
				left = p + 1
			}
		} else if arr[p] > target {
			if arr[left] > arr[right] { // 左边大于右边时，调整搜索方向
				left = p + 1
			} else {
				right = p - 1
			}
		}
	}
	return -1
}
