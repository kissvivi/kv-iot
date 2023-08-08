package main

import (
	"fmt"
	"testing"
)

func Test_taskFlow(t *testing.T) {
	i := 0
	link := NewLinkList()
	for {
		i++

		tf := taskFlow{}
		ts := taskStatus{}
		ts.setTaskStatus(fmt.Sprintf("demo:%d", i), "")
		a := action{}
		a.setAction("for", 100)
		tf.input = ts
		tf.action = a

		link.RPush(&tf)
		if i == 5 {
			break
		}
	}

	linkList := link.Range(0, link.Len())
	for _, node := range linkList {
		fmt.Printf("%v->%+v\n", node, node.value)
	}

}
