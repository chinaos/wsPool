// bill 2018.1.8
//优先级队列[同级别先进先出]权重值越大越优先
package queue

import (
	"container/list"
	"log"
	"sync"
	"time"
)

type Item struct {
	Data         interface{} //数据
	Priority     int32         //优先级
	AddTime   time.Time      //插入队列的时间
	Expiration int64 //过期时间值 以秒为单位
}

type PriorityQueue struct {
	Data        *list.List
	PriorityMap map[int32]*pqmap
}

type pqmap struct {
	element *list.Element
	totle   int
}

var lock sync.RWMutex

func NewPriorityQueue() *PriorityQueue {
	pq:= &PriorityQueue{
		Data:        list.New(),
		PriorityMap: make(map[int32]*pqmap),
	}

	return pq
}

func (pq *PriorityQueue) Len() int {
	defer lock.RUnlock()
	lock.RLock()
	return pq.Data.Len()
}

func (pq *PriorityQueue) Push(v *Item) {
	defer lock.Unlock()
	lock.Lock()

	newElement := pq.Data.PushFront(v)
	if _, ok := pq.PriorityMap[v.Priority]; !ok {
		pq.PriorityMap[v.Priority] = &pqmap{
			element: newElement,
			totle:   1,
		}
	} else {
		pq.PriorityMap[v.Priority].totle = pq.PriorityMap[v.Priority].totle + 1
	}
	//找出小于自己的最大值权重值
	var maxKey int32 = 1
	for p, _ := range pq.PriorityMap {
		if p < v.Priority && p >= maxKey {
			maxKey = p
		}
	}
	//pq.Dump()
	if v.Priority != maxKey {
		if _, ok := pq.PriorityMap[maxKey]; ok {
			pq.Data.MoveAfter(newElement, pq.PriorityMap[maxKey].element)
		}
	}
	//log.Println("挺入队列的消息：",v,"消息权重值：",v.Priority)
}

func (pq *PriorityQueue) Pop() *Item {
	defer lock.Unlock()
	lock.Lock()
	iter := pq.Data.Back()
	if iter==nil||iter.Value==nil{
		return nil
	}
	v := iter.Value.(*Item)
	pq.Data.Remove(iter)
	if pq.PriorityMap[v.Priority].totle > 1 {
		pq.PriorityMap[v.Priority].totle = pq.PriorityMap[v.Priority].totle - 1
	} else {
		delete(pq.PriorityMap, v.Priority)
	}
	//log.Println("取出队列的消息：",v,"消息权重值：",v.Priority)
	return v
}

func (pq *PriorityQueue) Dump() {
	for iter := pq.Data.Back(); iter != nil; iter = iter.Prev() {
		log.Println("队列信息:", iter.Value.(*Item))
	}
}
//清除队列
func (pq *PriorityQueue) Clear() {
	defer lock.RUnlock()
	lock.RLock()
	for iter := pq.Data.Back(); iter != nil; iter = iter.Prev() {
		//fmt.Println("item:", iter.Value.(*Item))
		v := iter.Value.(*Item)
		pq.Data.Remove(iter)
		if pq.PriorityMap[v.Priority].totle > 1 {
			pq.PriorityMap[v.Priority].totle = pq.PriorityMap[v.Priority].totle - 1
		} else {
			delete(pq.PriorityMap, v.Priority)
		}
	}
}

/*检测超时任务*/
func (pq *PriorityQueue) Expirations(expriCallback func(item *Item)) {
	defer lock.RUnlock()
	lock.RLock()
	for iter := pq.Data.Back(); iter != nil; iter = iter.Prev() {
		//fmt.Println("item:", iter.Value.(*Item))
		v := iter.Value.(*Item)
		if v.Expiration==0 {
			continue
		}
		isExpri:=v.AddTime.Add(time.Duration(v.Expiration)*time.Second).Before(time.Now())
		if isExpri {
			pq.Data.Remove(iter)
			if pq.PriorityMap[v.Priority].totle > 1 {
				pq.PriorityMap[v.Priority].totle = pq.PriorityMap[v.Priority].totle - 1
			} else {
				delete(pq.PriorityMap, v.Priority)
			}
			expriCallback(v)
		}else{
			//没过期说明越往前越新
			//break
		}

	}
}
