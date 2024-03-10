package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

func getRandomIndex(numOfelements int, questionRecord *map[int]interface{}) int {
	var randomindex int
	for {
		// Create a new random number generator with a custom seed (e.g., current time)
		source := rand.NewSource(time.Now().UnixNano())
		rng := rand.New(source)

		// if numOfelements == 10, it generates a random number of index between 0 and 9
		randomindex = rng.Intn(numOfelements)
		if _, ok := (*questionRecord)[randomindex]; !ok {
			break
		}
	}

	return randomindex
}

func getQuestions(category *map[string][]string, questionRecord *map[int]interface{}) (string, string) {
	// map[string][]string
	// 1. 依照category找出相應的slice  2. random出一個index，並檢查是否出現在"已使用"的set裡面，如果有就重抽
	// 再random另外一個index，這兩個index不得重複，並檢查是否出現在"已使用"的set裡面
	// 之後要把這裡的"動物"改成一個categoryName的變數
	numOfelements := len((*category)["日常"])
	var l, r int
	l = getRandomIndex(numOfelements, questionRecord)
	for {
		r = getRandomIndex(numOfelements, questionRecord)
		if l != r {
			break
		}
	}
	fmt.Println((*category)["日常"][l], (*category)["日常"][r])

	return (*category)["日常"][l] + "@" + strconv.Itoa(l), (*category)["日常"][r] + "@" + strconv.Itoa(r)
}
