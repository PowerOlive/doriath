package main

import (
	"log"
	"math/rand"
	"sort"
)

func main() {
	for SIZE := 100; SIZE <= 10000000; SIZE *= 10 {
		newTree := &node{
			key: rand.Uint64(),
		}
		var keys []uint64
		var lengths []int
		var sum int
		for i := 0; i < SIZE; i++ {
			newkey := rand.Uint64()
			keys = append(keys, newkey)
			newTree.insert(newkey)
			lgth := len(newTree.search(newkey))
			lengths = append(lengths, lgth)
			sum += lgth
		}
		sort.Ints(lengths)
		log.Println("size = ", SIZE,
			"\tmedian =", lengths[len(lengths)/2]*484,
			"\tupper =", lengths[len(lengths)*95/100]*484,
			"\tlower =", lengths[len(lengths)*5/100]*484)
	}
}
