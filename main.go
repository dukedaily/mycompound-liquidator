package main

import (
	// "liquidator/log"

	"fmt"
	"liquidator/conf"
	"liquidator/contract"
	"liquidator/executor"
	"liquidator/handler"
	"liquidator/log"
	"os"
	"os/signal"
	"syscall"
)

func initLog() {
	fileDir := conf.Config.Log.FileDir
	fileName := conf.Config.Log.FileName
	prefix := conf.Config.Log.Prefix
	level := conf.Config.Log.Level

	if err := log.Init(fileDir, fileName, prefix, level); err != nil {
		panic(err)
	}
}

func init() {
	conf.Init()

	initLog()

	contract.Init()
}

func main() {
	fmt.Println("starting...")
	handler.Start()
	go executor.Run()

	//如果监听到系统信号 SIGQUIT 就退出程序，否则一直阻塞
	exitChan := make(chan int)
	signalChan := make(chan os.Signal, 1)
	go func() {
		<-signalChan
		log.Print("Received signal SIGQUIT")
		exitChan <- 1
	}()
	signal.Notify(signalChan, syscall.SIGQUIT)
	<-exitChan
}
