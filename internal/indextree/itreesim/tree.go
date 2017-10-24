package main

type node struct {
	key   uint64
	left  *node
	right *node
}

func (nd *node) insert(key uint64) {
	path := nd.search(key)
	tip := path[len(path)-1]
	if key < tip.key {
		tip.left = &node{key: key}
	} else if key > tip.key {
		tip.right = &node{key: key}
	}
	return
}

func (nd *node) search(key uint64) (path []*node) {
	path = append(path, nd)
	for {
		tip := path[len(path)-1]
		var next *node
		if key < tip.key {
			next = tip.left
		} else if key > tip.key {
			next = tip.right
		}
		if next == nil {
			return
		}
		path = append(path, next)
	}
}
