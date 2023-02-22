package main

import "fmt"

func main() {
	s := "abcd你好"
	fmt.Println(s)
	var sss rune
	for _, ss := range s {
		sss = ss
		fmt.Println(string(sss))
	}
	//ss := []rune(s)
	//for i := 0; i < len(ss); i++ {
	//	fmt.Println(string(ss[i]))
	//}
}
