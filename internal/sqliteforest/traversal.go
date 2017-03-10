package sqliteforest

import (
	"database/sql"
	"sort"
	"strings"
)

// search for a value given a key in a (sub)tree, returning a path leading to the "closest" part of the tree; a path is returned regardless of whether or not the key actually exists
func searchTree(root cursor, key string) (path []cursor, err error) {
	for !root.isNull() {
		var rec fullNode
		rec, err = root.getRecord()
		if err != nil {
			return
		}
		path = append(path, root)
		switch strings.Compare(key, rec.Key) {
		case -1:
			root, err = root.getLeft()
			if err != nil {
				return
			}
		case 1:
			root, err = root.getRight()
			if err != nil {
				return
			}
		default:
			return
		}
	}
	return
}

// allocate an entire dictionary into a new tree, returning the root
func allocDict(tx *sql.Tx, dict map[string][]byte) (root cursor, err error) {
	// sort the dict keys into a slice
	var skeys []string
	for k := range dict {
		skeys = append(skeys, k)
	}
	sort.Strings(skeys)
	// algorithm to convert sorted list to perfectly balanced BST
	var rec func([]string) (cursor, error)
	rec = func(sl []string) (cursor, error) {
		if len(sl) == 0 {
			return cursor{}, nil
		} else if len(sl) == 1 {
			return allocCursor(tx,
				fullNode{
					Key:   sl[0],
					Value: dict[sl[0]]})
		} else {
			left := sl[:len(sl)/2]
			mid := sl[len(sl)/2]
			right := sl[len(sl)/2+1:]
			lcurs, err := rec(left)
			if err != nil {
				return cursor{}, err
			}
			rcurs, err := rec(right)
			if err != nil {
				return cursor{}, err
			}
			return allocCursor(tx,
				fullNode{Key: mid, Value: dict[mid],
					LeftHash: lcurs.loc, RightHash: rcurs.loc})
		}
	}
	return rec(skeys)
}
