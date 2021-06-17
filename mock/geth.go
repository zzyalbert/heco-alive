package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	fmt.Printf("geth start with %s\n", strings.Join(os.Args, " "))
	println(os.Args[0])

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGTERM, syscall.SIGINT)

	t := time.NewTicker(time.Second)
	i := 0
	for {
		select {
		case s := <-sigc:
			fmt.Println(fmt.Sprintf("killed by signal: %v", s))
			time.Sleep(time.Second * 15)
			os.Exit(0)
		case <-t.C:
			fmt.Printf("geth is running [%d]\n", i)
			i++
		default:
		}
	}
}
