// Copyright 2017-2019 gf Author(https://github.com/gogf/gf). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Package grpool implements a goroutine reusable pool.
package wsPool

import (
	"container/list"
	"errors"
)

// Goroutine Pool
type Pool struct {
	limit  int         // Max goroutine count limit.
	count  int  // Current running goroutine count.
	list   *list.List // Job list for asynchronous job adding purpose.
	closed bool // Is pool closed or not.
	wJobsChan chan func() //写工作方法
	rJobsChan chan func() //读工作方法
}



// New creates and returns a new goroutine pool object.
// The parameter <limit> is used to limit the max goroutine count,
// which is not limited in default.
func New(limit ...int) *Pool {
	p := &Pool{
		limit:  -1,
		count:  0,
		list:   list.New(),
		closed: false,
		wJobsChan:make(chan func()),
		rJobsChan:make(chan func()),
	}
	if len(limit) > 0 && limit[0] > 0 {
		p.limit = limit[0]
	}
	//开始工作
	go p.runWrite()
	go p.runRead()
	return p
}

/*
// Default goroutine pool.
var pool = New()
// Add pushes a new job to the pool using default goroutine pool.
// The job will be executed asynchronously.
func Add(f func()) error {
	return pool.Add(f)
}

// Size returns current goroutine count of default goroutine pool.
func Size() int {
	return pool.Size()
}

// Jobs returns current job count of default goroutine pool.
func Jobs() int {
	return pool.Jobs()
}
*/


func (p *Pool) runWrite(){
	for !p.closed  {
		select {
		case f,ok:=<-p.wJobsChan:
			if !ok {
				break
			}
			p.list.PushFront(f)
		}
	}
}

func (p *Pool) runRead(){
	for !p.closed  {
		if job := p.list.Back(); job != nil {
			value := p.list.Remove(job)
			p.rJobsChan<-value.(func())
		} else {
			return
		}
	}
}

// Add pushes a new job to the pool.
// The job will be executed asynchronously.
func (p *Pool) Add(f func()) error {
	for p.closed{
		return errors.New("pool closed")
	}
	p.wJobsChan<-f

	var n int
		n = p.count
	if p.limit != -1 && n >= p.limit {
		return nil
	}
	p.count=n+1
	p.fork()
	return nil
}

// Cap returns the capacity of the pool.
// This capacity is defined when pool is created.
// If it returns -1 means no limit.
func (p *Pool) Cap() int {
	return p.limit
}

// Size returns current goroutine count of the pool.
func (p *Pool) Size() int {
	return p.count
}

// Jobs returns current job count of the pool.
func (p *Pool) Jobs() int {
	return p.list.Len()
}

// fork creates a new goroutine pool.
func (p *Pool) fork() {
	go func() {
		defer func() {
			p.count--
		}()
		for !p.closed {
			select {
			case job,ok:=<-p.rJobsChan:
				if !ok {
					break
				}
				job()
			}
		}
	}()
}

// IsClosed returns if pool is closed.
func (p *Pool) IsClosed() bool {
	return p.closed
}

// Close closes the goroutine pool, which makes all goroutines exit.
func (p *Pool) Close() {
	p.closed=true
	close(p.wJobsChan)
	close(p.rJobsChan)
}
