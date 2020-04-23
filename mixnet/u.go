package main

import (
	"fmt"
	"time"
)

type i struct{
	a int
	b int
}

func b(a []i){
	boom := time.After(500 * time.Millisecond)
	<-boom
	fmt.Println(a)
}

func main() {
	a := []i{{1,2},{3,5}}
	go b(a)
	a = []i{}
	boom := time.After(700 * time.Millisecond)
	<-boom
	fmt.Println(a)
}
