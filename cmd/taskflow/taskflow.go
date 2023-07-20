package main

type taskStatus struct {
	name   string
	status string
}

type action struct {
	aType string
	delay int //延时 配合 循环使用
}

type taskFlow struct {
	input  taskStatus //输入
	output taskStatus //输出
	step   int        //步数
	action action     //动作  循环 for 判断 if else
}

func (ts *taskStatus) setTaskStatus(name, status string) {
	ts.status = status
	ts.name = name
}

func (ts *taskStatus) getTaskStatus() (name, status string) {
	return ts.name, ts.status
}

func (a *action) setAction(aType string, delay int) {
	a.aType = aType
	a.delay = delay
}

func (a *action) getAction() (aType string, delay int) {
	return a.aType, a.delay
}
