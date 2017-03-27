package sqliteforest

import "sync"

var reccache struct {
	table map[string]fullNode
	delqu []string
	lock  sync.Mutex
}

func rcGet(key string, gen func() (fullNode, error)) (fullNode, error) {
	LIMIT := 1024 * 1024
	reccache.lock.Lock()
	defer reccache.lock.Unlock()
	res, ok := reccache.table[key]
	if ok {
		return res, nil
	}
	res, err := gen()
	if err != nil {
		return fullNode{}, err
	}
	reccache.table[key] = res
	reccache.delqu = append(reccache.delqu, key)
	if len(reccache.delqu) > LIMIT {
		delete(reccache.table, reccache.delqu[0])
		reccache.delqu = reccache.delqu[1:]
	}
	return res, nil
}

func init() {
	reccache.table = make(map[string]fullNode)
}
