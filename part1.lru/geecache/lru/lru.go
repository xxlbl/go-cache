package lru

import "container/list"

// Cache is a LRU cache. It is not safe for concurrent access.
type Cache struct {
	maxBytes int64                    // 允许使用的最大内存
	nbytes   int64                    // 当前已使用的内存
	ll       *list.List               // 标准库双向链表
	cache    map[string]*list.Element // k：字符串，v：双向链表节点指针
	// optional and executed when an entry is purged.
	OnEvicted func(key string, value Value) //某条记录被移除时的回调函数，可以为 nil。
}

//双向链表节点的数据类型，
//在链表中仍保存每个值对应的 key 的好处在于，淘汰队首节点时，需要用 key 从字典中删除对应的映射。
type entry struct {
	key   string
	value Value
}

// Value use Len to count how many bytes it takes
//值是实现了 Value 接口的任意类型，该接口只包含了一个方法 Len() int，用于返回值所占用的内存大小。
type Value interface {
	Len() int
}

// New is the Constructor of Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// Add adds a value to the cache.
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		// 如果键存在，则更新对应节点的值，并将该节点移到队尾。
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		// 更新长度
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		// 不存在则新增，首先队尾添加新节点, 并字典中添加 key 和节点的映射关系。
		ele := c.ll.PushFront(&entry{key, value})
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	//更新 c.nbytes，如果超过了设定的最大值 c.maxBytes，则移除最少访问的节点。
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

// Get look ups a key's value
//查找主要有 2 个步骤，第一步是从字典中找到对应的双向链表的节点，第二步，将该节点移动到队尾
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		//如果键对应的链表节点存在，则将对应节点移动到队尾，并返回查找到的值。在这里约定 front 为队尾
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

// RemoveOldest removes the oldest item
// 缓存淘汰,移除最近最少访问的节点（队首）
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back() // c.ll.Back() 取到队首节点，从链表中删除。

	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key) // 从字典中 c.cache 删除该节点的映射关系。
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Len the number of cache entries
func (c *Cache) Len() int {
	return c.ll.Len()
}
