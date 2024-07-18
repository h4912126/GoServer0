package main

import (
	"fmt"
	"time"
)

// 引入自定义的Server模块

// 定义一个启动Server的函数
// main.go 脚本

func StartServer() {
	server := NewServer("[::]", 9527)
	server.Start()
}

// main.go 脚本

func main() {
	// 启动Server
	go StartServer()

	// TODO 你可以写其他逻辑
	fmt.Println("这是一个Go服务端，实现了Socket消息广播功能")

	// 防止主线程退出
	for {
		time.Sleep(1 * time.Second)
	}
}
