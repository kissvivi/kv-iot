package main

// LinkNode 链表节点结构
type LinkNode struct {
	value *taskFlow
	prev  *LinkNode //前驱
	next  *LinkNode //构成链表 后驱

}

// NewLinkNode 创建一个节点
func NewLinkNode(value *taskFlow) *LinkNode {
	return &LinkNode{value: value}
}

// Prev 当前节点的前一个节点
func (n *LinkNode) Prev() (prev *LinkNode) {
	prev = n.prev
	return
}

// Next 当前节点的后一个节点
func (n *LinkNode) Next() (next *LinkNode) {
	next = n.next
	return
}

// GetValue 获取当前节点的值
func (n *LinkNode) GetValue() (value *taskFlow) {
	if n == nil {
		return
	}
	value = n.value
	return
}
