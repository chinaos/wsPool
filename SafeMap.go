package wsPool


type SafeMap struct {
	//Lock *sync.RWMutex

	m map[*Client]bool
}

//创建线程安全的map
func NewSafeMap() *SafeMap {

	return &SafeMap{

		//Lock: new(sync.RWMutex),

		m: make(map[*Client]bool),
	}

}

//如果不存在，则返回nil
func (sm *SafeMap) Get(k *Client) bool {

	//sm.Lock.RLock()

	//defer sm.Lock.RUnlock()

	if val, ok := sm.m[k]; ok {
		return val
	}
	return false

}

//如果不存在，则返回nil
func (sm *SafeMap) GetAll() map[*Client]bool {
	return sm.m
}

//Key不存在时才修改
func (sm *SafeMap) Set(k *Client, v bool) bool {

	//sm.Lock.Lock()

	//defer sm.Lock.Unlock()

	sm.m[k] = v

	return true

}

//如果存在则返回TRUE
func (sm *SafeMap) Check(k *Client) bool {

	//sm.Lock.RLock()

	//defer sm.Lock.RUnlock()

	if _, ok := sm.m[k]; !ok {

		return false

	}

	return true

}

func (sm *SafeMap) Delete(k *Client) {

	//sm.Lock.Lock()

	//defer sm.Lock.Unlock()

	delete(sm.m, k)

}

// Iterator iterates the hash map with custom callback function <f>.
// If <f> returns true, then it continues iterating; or false to stop.
func (sm *SafeMap) Iterator(f func(k *Client, v  bool) bool) {
	//sm.Lock.Lock()
	//defer sm.Lock.Unlock()
	for k, v := range sm.m {
		if !f(k, v) {
			break
		}
	}
}
