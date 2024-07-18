package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
)

// user.go 脚本

// user.go 脚本

type User struct {
	Name    string      // 昵称，默认与Addr相同
	Addr    string      // 地址
	Channel chan string // 消息管道
	conn    net.Conn    // 连接
	server  *Server     // 缓存Server的引用
}

func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()

	user := &User{
		Name:    userAddr,
		Addr:    userAddr,
		Channel: make(chan string),
		conn:    conn,
		server:  server,
	}
	go user.ListenMessage()
	return user
}

// user.go 脚本

func (user *User) Online() {

	// 用户上线，将用户加入到OnlineMap中，注意加锁操作
	user.server.mapLock.Lock()
	user.server.OnlineMap[user.Name] = user
	user.server.mapLock.Unlock()

	// 广播当前用户上线消息
	user.server.BroadCast(user, "上线啦O(∩_∩)O")
}

// user.go 脚本

func (user *User) Offline() {

	// 用户下线，将用户从OnlineMap中删除，注意加锁
	user.server.mapLock.Lock()
	delete(user.server.OnlineMap, user.Name)
	user.server.mapLock.Unlock()

	// 广播当前用户下线消息
	user.server.BroadCast(user, "下线了o(╥﹏╥)o")
}

// user.go 脚本

func (user *User) DoMessage(buf []byte, len int) {
	//提取用户的消息(去除'\n')
	msg := string(buf[:len-1])
	// 调用Server的BroadCast方法
	user.server.BroadCast(user, msg)
}

func (user *User) ListenMessage() {
	for {
		msg := <-user.Channel
		fmt.Println("Send msg to client: ", msg, ", len: ", int16(len(msg)))
		bytebuf := bytes.NewBuffer([]byte{})
		// 前两个字节写入消息长度
		binary.Write(bytebuf, binary.BigEndian, int16(len(msg)))
		// 写入消息数据
		binary.Write(bytebuf, binary.BigEndian, []byte(msg))
		// 发送消息给客户端
		user.conn.Write(bytebuf.Bytes())
	}
}
