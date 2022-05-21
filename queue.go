/**
**队列模块。使用channel作为存放数据的存储
 */
package navi_go_log

import (
	"time"
)

type queue struct {
	value   chan interface{}
	maxSize int
	timeout time.Duration
}

// 获取记录
func (q *queue) Get() (interface{}, bool) {
	select {
	case v, ok := <-q.value:
		if !ok {
			return nil, false
		}
		return v, true

	case <-time.After(q.timeout):
		return nil, false
	}
}

// 放入队列
func (q *queue) Put(v interface{}) bool {
	select {
	case q.value <- v:
		return true

	case <-time.After(q.timeout):
		return false
	}
}

// 获取队列大小
func (q *queue) Size() int {
	return len(q.value)
}

// 判断队列是否为空
func (q *queue) Empty() bool {
	return len(q.value) == 0
}

// 判断队列是否已满
func (q *queue) Full() bool {
	return len(q.value) == cap(q.value)
}

// 关闭队列通道
func (q *queue) Close() {
	close(q.value)
}

// 创建队列
func NewQueue(maxSize int, timeout time.Duration) *queue {
	queue := queue{
		value:   make(chan interface{}, maxSize),
		maxSize: maxSize,
		timeout: timeout,
	}
	return &queue
}
