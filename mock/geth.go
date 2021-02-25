package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	fmt.Printf("geth start with %s\n", strings.Join(os.Args, " "))
	println(os.Args[0])

	t := time.NewTicker(time.Second)
	i := 0
	for {
		select {
		case <-t.C:
			fmt.Printf("geth is running [%d]\n", i)
			i++
		default:
		}
	}
}
