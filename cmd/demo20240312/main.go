package main

func main() {
	majorityElement([]int{2, 3, 4, 5, 4})
}

func majorityElement(nums []int) int {

	p := 1
	num := nums[0]
	v := 1
	for p < len(nums) {

		if v == 0 {
			num = nums[p]
		} else if num == nums[p] {
			v++
		} else {
			v--
		}
		p++
	}
	return num
}
