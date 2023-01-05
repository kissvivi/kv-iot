package utils

import (
	"fmt"
	"strconv"
	"time"
)

type GenID struct {
	DateString string
	NO         string
}

var _no int

func NewGenID() *GenID {
	_no++
	return &GenID{DateString: time.Now().Format("2006-0102-1504-05"), NO: strconv.Itoa(_no)}
}

func (t GenID) String() string {
	return fmt.Sprintf("%s-%s", t.DateString, t.NO)
}
