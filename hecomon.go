package main

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"gopkg.in/natefinch/lumberjack.v2"
	"log"
	"math/big"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	rpcHost       = "http://127.0.0.1:8545"
	fetchInterval = 3
	killCount     = 100
)

type State struct {
	height uint64
	block  *types.Block
	count  int64
}

func (m State) String() string {
	return fmt.Sprintf("height: %d, count:%d", m.height, m.count)
}

var (
	lastState *State
	childCmd  *exec.Cmd
	args      []string
)

func main() {
	fmt.Printf("hecomon start with %s\n", strings.Join(os.Args, " "))

	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s [command] \n", os.Args[0])
		os.Exit(1)
	}
	args = os.Args

	log.SetOutput(&lumberjack.Logger{
		Filename:   "./hecomon.log",
		MaxSize:    10, // megabytes
		MaxBackups: 4,
		MaxAge:     28,    //days
		Compress:   false, // disabled by default
	})

	go monitorLoop()

	for {
		runApp()
	}
}

func runApp() {
	log.Printf("start command with args %s", strings.Join(args[1:], " "))

	childCmd = exec.Command(args[1], args[2:]...)
	childCmd.Stdout = os.Stdout
	childCmd.Stderr = os.Stderr
	childCmd.Stdin = os.Stdin

	if err := childCmd.Run(); err != nil {
		log.Printf("error in start %s \n", err)
	}

	//clear state
	clearState()
}

func monitorLoop() {
	t := time.NewTicker(fetchInterval * time.Second)

	for {
		select {
		case <-t.C:
			fetchBlock()
		}
	}
}

func fetchBlock() {
	rpcClient, err := ethclient.Dial(rpcHost)
	if err != nil {
		log.Printf("dial host %s %s\n", rpcHost, err)
		handleErr()
		return
	}

	log.Println("fetch block at", time.Now())

	height, err := rpcClient.BlockNumber(context.Background())
	if err != nil || height == 0 {
		log.Printf("erro in block height, %s \n", err)
		handleErr()
		return
	}

	log.Println("read height:", height)

	block, err := rpcClient.BlockByNumber(context.Background(), new(big.Int).SetUint64(height))
	if err != nil || block == nil {
		log.Printf("erro in block %s, \n ", err)
		handleErr()
		return
	}

	log.Printf("read block %d -> %s: \n", block.Header().Number, block.Hash().String())

	//count
	count(height, block)
}

func count(height uint64, block *types.Block) {
	if lastState == nil {
		lastState = &State{height, block, 0}
	} else {
		//count stuck
		if height <= lastState.height {
			lastState.count++
			printStatus()
		} else {
			lastState.height = height
			lastState.count = 0
			printStatus()
		}
	}

	tryKill()
}

func tryKill() {
	//trigger kill

	printStatus()

	if lastState != nil && lastState.count >= killCount {
		ret := kill()
		if ret {
			clearState()
		}
	}
}

func clearState() {
	lastState = nil
	log.Println("clear state")
}

func kill() bool {
	err := childCmd.Process.Kill()
	if err != nil {
		log.Printf("kill error %s \n", err)
	}

	return err == nil
}

func handleErr() {
	if lastState == nil {
		lastState = &State{0, nil, 0}
	} else {
		lastState.count++
	}

	tryKill()
}

func printStatus() {
	log.Printf("now: %s, state:%s ", time.Now().String(), lastState)
}
