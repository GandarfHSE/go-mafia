package algo

import (
	"log"
	"math/rand"
	"time"
)

func Erase[T any](a []T, ind int) []T {
	if ind >= len(a) {
		log.Fatal("Erase: wrong index")
	}

	a[ind] = a[len(a)-1]
	return a[:len(a)-1]
}

// copypasted from https://yourbasic.org/golang/shuffle-slice-array/
func Shuffle[T any](a []T) []T {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(a), func(i, j int) { a[i], a[j] = a[j], a[i] })
	return a
}
