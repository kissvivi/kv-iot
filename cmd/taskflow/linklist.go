package main

// LinkList 链表
type LinkList struct {
	head *LinkNode // 表头节点
	tail *LinkNode // 表尾节点
	len  int       // 链表的长度
}

// NewLinkList 创建一个空链表
func NewLinkList() *LinkList {
	return &LinkList{}
}

// Head 返回链表头节点
func (l *LinkList) Head() (head *LinkNode) {
	head = l.head
	return
}

// Tail 返回链表尾节点
func (l *LinkList) Tail() (tail *LinkNode) {
	tail = l.tail
	return
}

// Len 返回链表长度
func (l *LinkList) Len() (len int) {
	len = l.len
	return
}

// RPush 在链表的右边插入一个元素
func (l *LinkList) RPush(value *taskFlow) {

	node := NewLinkNode(value)

	// 链表空的时候
	if l.Len() == 0 {
		l.head = node
		l.tail = node
	} else {
		tail := l.tail
		tail.next = node
		node.prev = tail

		l.tail = node
	}

	l.len = l.len + 1

	return
}

// LPop 从链表左边取出一个节点
func (l *LinkList) LPop() (node *LinkNode) {

	// 数据为空
	if l.len == 0 {

		return
	}
	node = l.head
	if node.next == nil {
		// 链表未空
		l.head = nil
		l.tail = nil
	} else {
		l.head = node.next
	}
	l.len = l.len - 1

	return
}

// Index 通过索引查找节点
// 查不到节点则返回空
func (l *LinkList) Index(index int) (node *LinkNode) {

	// 索引为负数从表尾开始查找
	if index < 0 {
		index = (-index) - 1
		node = l.tail
		for true {
			// 未找到
			if node == nil {

				return
			}

			// 查到数据
			if index == 0 {

				return
			}

			node = node.prev
			index--
		}
	} else {
		node = l.head
		for ; index > 0 && node != nil; index-- {
			node = node.next
		}
	}

	return
}

// Range 返回指定区间的元素
func (l *LinkList) Range(start, stop int) (nodes []*LinkNode) {
	nodes = make([]*LinkNode, 0)

	// 转为正数
	if start < 0 {
		start = l.len + start
		if start < 0 {
			start = 0
		}
	}

	if stop < 0 {
		stop = l.len + stop
		if stop < 0 {
			stop = 0
		}
	}

	// 区间个数
	rangeLen := stop - start + 1
	if rangeLen < 0 {

		return
	}

	startNode := l.Index(start)
	for i := 0; i < rangeLen; i++ {
		if startNode == nil {
			break
		}

		nodes = append(nodes, startNode)
		startNode = startNode.next
	}

	return
}
