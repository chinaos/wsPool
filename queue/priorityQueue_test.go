package queue

import (
	"container/list"
	"fmt"
	"log"
	"math/rand"
	"testing"
	"time"
)

var TQueues *PriorityQueue

func TestNewPriorityQueue(t *testing.T) {
	TQueues = NewPriorityQueue(func(item *Item) {
		log.Println("已经过期的节点：",gconv.String(item))
	})
	//pq := NewPriorityQueue(func(item *Item) {
	//	log.Println("已经过期的节点：",gconv.String(item))
	//})
	TQueues.Push(&Item{
		Priority: 9,
		Data:     "优先级为9的消息,",

	})
	TQueues.Push(&Item{
		Priority: 11,
		Data:     "优先级为11的消息,",
	})
	TQueues.Push(&Item{
		Priority: 8,
		Data:     "优先级为8的消息,",
	})
	TQueues.Push(&Item{
		Priority: 10,
		Data:     "优先级为10的消息,",

	})
	TQueues.Push(&Item{
		Priority: 13,
		Data:     "优先级为13的消息,",

	})
	TQueues.Push(&Item{
		Priority: 15,
		Data:     "优先级为15的消息,",

	})
	TQueues.Push(&Item{
		Priority: 11,
		Data:     "优先级为11的消息,",

	})
	TQueues.Push(&Item{
		Priority: 7,
		Data:     "优先级为7的消息,",
	})
	TQueues.Push(&Item{
		Priority: 7,
		Data:     "优先级为7的消息",
	})
	TQueues.Push(&Item{
		Priority: 7,
		Data:     "优先级为7的消息",
	})
	TQueues.Push(&Item{
		Priority: 7,
		Data:     "优先级为7的消息,",
	})
	TQueues.Push(&Item{
		Priority: 7,
		Data:     "优先级为7的消息,",
	})
	TQueues.Push(&Item{
		Priority: 7,
		Data:     "优先级为7的消息,",
	})
	TQueues.Push(&Item{
		Priority: 6,
		Data:     "优先级为6的消息",
	})
	TQueues.Push(&Item{
		Priority: -1,
		Data:     "优先级为-1的消息,",
	})

	for TQueues.Len() > 0 {
		v := TQueues.Pop()
		fmt.Println(v.Data)
		time.Sleep(5 * time.Second)

	}
}

type JieGou struct {
	Content string
	Priority int
}

type Ks struct {
	list *list.List
	maps map[int]*Priority
}

type Priority struct {
	element *list.Element
}

func (k *Ks)NewPushBack(v JieGou)  {
	if k.list.Len() == 0 {
		insertEl := k.list.PushFront(v)
		k.maps[v.Priority] = &Priority{
			element: insertEl,
		}
	}else {
		vvF, ok := k.list.Front().Value.(JieGou)
		if ok {
			if vvF.Priority < v.Priority {
				insertEl := k.list.PushFront(v)
				k.maps[v.Priority] = &Priority{
					element: insertEl,
				}
				return
			}
		}
		vvB, ok := k.list.Back().Value.(JieGou)
		if ok {
			if vvB.Priority >= v.Priority {
				insertEl := k.list.PushBack(v)
				k.maps[v.Priority] = &Priority{
					element: insertEl,
				}
				return
			}
		}
		maxKey := 0
		for kk,_ := range k.maps{
			if kk <= v.Priority && kk >= maxKey {
				maxKey = kk
			}
		}
		if _, ok := k.maps[maxKey]; ok {
			insertEl := k.list.PushBack(v)
			k.list.MoveAfter(insertEl, k.maps[maxKey].element)
			k.maps[v.Priority] = &Priority{
				element: insertEl,
			}
		}
	}
}

func TestPriorityQueue_Pop(t *testing.T) {
	var ks = &Ks{
		list.New(),
		make(map[int]*Priority),
	}
	//var kkone = new(list.List)
	//PushBack 放到list的最后面
	//PushFront 放到list的最前面
	//Remove 仅仅是将Element对象移除的方法，没有一定是先移除front或者back，是业务自己定义移除的Element
	//理论上要不就是从最后开始移除，或者从最前面开始移除
	//MoveAfter 两个list调换位置
	ks.NewPushBack(JieGou{
		Content:"a",
		Priority:20,
	})
	ks.NewPushBack(JieGou{
		Content:"b",
		Priority:30,
	})
	ks.NewPushBack(JieGou{
		Content:"c",
		Priority:40,
	})
	ks.NewPushBack(JieGou{
		Content:"d",
		Priority:50,
	})
	ks.NewPushBack(JieGou{
		Content:"e",
		Priority:40,
	})
	ks.NewPushBack(JieGou{
		Content:"f",
		Priority:30,
	})
	ks.NewPushBack(JieGou{
		Content:"g",
		Priority:20,
	})
	for ks.list.Len() > 0  {
		useMsg := ks.list.Front()
		log.Println(useMsg)
		ks.list.Remove(ks.list.Front())
	}
}
