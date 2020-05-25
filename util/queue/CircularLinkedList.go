package queue

//环形队列
import (
	"container/list"
	"fmt"
	"sync"
)

type CircularLinked struct {
	data *list.List
}

var lock1 sync.Mutex

var _ *CircularLinked

func (q *CircularLinked) Push(v interface{}) {
	defer lock1.Unlock()
	lock1.Lock()
	q.data.PushFront(v)
}

func (q *CircularLinked) Pop() interface{} {
	defer lock1.Unlock()
	lock1.Lock()
	iter := q.data.Back()
	v := iter.Value
	q.data.Remove(iter)
	return v
}

//读取第一个元素后,把它移到队列尾
func (q *CircularLinked) Front() *list.Element {
	defer lock1.Unlock()
	lock1.Lock()
	iter := q.data.Front()
	q.data.Remove(iter)
	if iter != nil {
		//q.data.MoveToBack(iter)
		q.data.PushBack(iter.Value.(string))
	}

	return iter
}

//移除元素
func (q *CircularLinked) Remove(iter *list.Element) {
	defer lock1.Unlock()
	lock1.Lock()
	if iter != nil {
		q.data.Remove(iter)
	}
}

func (q *CircularLinked) Len() int {
	return q.data.Len()
}

func (q *CircularLinked) Dump() {
	for iter := q.data.Back(); iter != nil; iter = iter.Prev() {
		fmt.Println("item:", iter.Value)
	}
}
