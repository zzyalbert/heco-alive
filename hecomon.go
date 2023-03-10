package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	path = "./hecomon_config.toml"
)

type Config struct {
	LogPath       string
	Rpc           string
	FetchInterval int
	KillCount     int
}

var defaultConfig Config = Config{
	LogPath:       "./logs/hecomon.log",
	Rpc:           "http://127.0.0.1:8545",
	FetchInterval: 3,
	KillCount:     40,
}

type State struct {
	height uint64
	block  *types.Header
	count  int64
}

func (m State) String() string {
	return fmt.Sprintf("height: %d, count:%d", m.height, m.count)
}

var (
	lastState  *State
	clearState chan int
	childCmd   *exec.Cmd
	args       []string
	config     *Config
)

func loadConfig() {

	config = &Config{}
	_, err := toml.DecodeFile(path, &config)
	if err != nil {
		log.Printf("read config error: %v", err)
	}

	if config.LogPath == "" {
		config.LogPath = defaultConfig.LogPath
	}

	if config.Rpc == "" {
		config.Rpc = defaultConfig.Rpc
	}

	if config.FetchInterval <= 0 {
		config.FetchInterval = defaultConfig.FetchInterval
	}

	if config.KillCount <= 0 {
		config.KillCount = defaultConfig.KillCount
	}

	log.Printf("config info: %v", config)
}

func main() {

	fmt.Printf("hecomon start with %s\n", strings.Join(os.Args, " "))

	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s [command] \n", os.Args[0])
		os.Exit(1)
	}
	args = os.Args

	//read config info
	loadConfig()

	log.SetOutput(&lumberjack.Logger{
		Filename:   "./logs/hecomon.log",
		MaxSize:    10, // megabytes
		MaxBackups: 4,
		MaxAge:     28,    //days
		Compress:   false, // disabled by default
	})

	clearState = make(chan int, 10)

	go monitorLoop()

	for {
		runApp()
		time.Sleep(1 * time.Second)
	}
}

func runApp() {
	log.Printf("start command with args %s", strings.Join(args[1:], " "))

	//clear state
	clearState <- 0

	childCmd = exec.Command(args[1], args[2:]...)
	childCmd.Stdout = os.Stdout
	childCmd.Stderr = os.Stderr
	childCmd.Stdin = os.Stdin
	if err := childCmd.Run(); err != nil {
		log.Printf("error in start %s \n", err)
	}
}

func monitorLoop() {
	t := time.NewTicker(time.Duration(config.FetchInterval) * time.Second)

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGTERM, syscall.SIGINT)

	for {
		select {
		case s := <-sigc:
			killWithSignal(s)
			os.Exit(0)

		case <-t.C:
			fetchBlock()
		case <-clearState:
			lastState = nil
			log.Println("clear state")
		}

		if childCmd != nil && childCmd.Process != nil {
			log.Printf("current child process :%v", childCmd.Process.Pid)
		}
	}
}

func getCurrentHeight(client *ethclient.Client) (height uint64, err error) {

	process, err := client.SyncProgress(context.Background())
	if err != nil {
		return
	}

	//syncing
	if process != nil {
		height = process.CurrentBlock
		return
	}

	height, err = client.BlockNumber(context.Background())
	if err != nil {
		return
	}

	return
}

func fetchBlock() {
	rpcClient, err := ethclient.Dial(config.Rpc)
	if err != nil {
		log.Printf("dial host %s %s\n", config.Rpc, err)
		handleErr()
		return
	}

	log.Println("fetch block at", time.Now())

	height, err := getCurrentHeight(rpcClient)
	if err != nil || height == 0 {
		log.Printf("erro in block height, %s \n", err)
		handleErr()
		return
	}

	log.Println("read height:", height)

	header, err := rpcClient.HeaderByNumber(context.Background(), new(big.Int).SetUint64(height))
	if err != nil || header == nil {
		log.Printf("erro in block %s, \n ", err)
		handleErr()
		return
	}

	log.Printf("read block %d -> %s: \n", header.Number, header.Hash().String())

	//count
	count(height, header)
}

func count(height uint64, header *types.Header) {
	if lastState == nil {
		lastState = &State{height, header, 0}
		return
	}

	//count stuck
	if height <= lastState.height {
		lastState.count++
		tryKill()
		return
	}

	lastState.height = height
	lastState.block = header
	lastState.count = 0
	printStatus()
}

func tryKill() {
	//trigger kill

	printStatus()

	if lastState != nil && lastState.count >= int64(config.KillCount) {
		log.Println("try kill....")
		err := killWithSignal(syscall.SIGINT)
		if err != nil {
			kill()
		}
	}
}

func killWithSignal(sig os.Signal) error {
	err := childCmd.Process.Signal(sig)
	if err != nil {
		log.Printf("kill with signal error %s \n", err)
	}

	clearState <- 0

	state, err := childCmd.Process.Wait()
	log.Printf("child process state: %v, err: %v", state, err)

	return err
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
