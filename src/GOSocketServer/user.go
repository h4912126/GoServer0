package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

// user.go 脚本

// user.go 脚本

type User struct {
	Name     string      // 昵称，默认与Addr相同
	Addr     string      // 地址
	userId   string      // 用户ID
	UserIcon string      // 用户头像
	Channel  chan string // 消息管道
	conn     net.Conn    // 连接
	server   *Server     // 缓存Server的引用
}

type UserInfo struct {
	Name     string `json:"name"`
	Addr     string `json:"addr"`
	UserId   string `json:"userId"`
	UserIcon string `json:"userIcon"`
}

type ChatRoomInfo struct {
	ChatRoomId      string `json:"chatRoomId"`
	ChatRoomName    string `json:"chatRoomName"`
	ChatRoomIcon    string `json:"chatRoomIcon"`
	ChatRoomContent string `json:"chatRoomContent"`
}

func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()

	user := &User{
		Name:     string('0'),
		Addr:     userAddr,
		UserIcon: "head_1",
		userId:   string('0'),
		Channel:  make(chan string),
		conn:     conn,
		server:   server,
	}
	go user.ListenMessage()
	return user
}

func SetUserId(user *User, userId string) {
	user.userId = userId
}

func SetUserName(user *User, name string) {
	user.Name = name
}
func SetUserIcon(user *User, icon string) {
	user.UserIcon = icon
}

// user.go 脚本

func (user *User) Online() {

	// 用户上线，将用户加入到OnlineMap中，注意加锁操作
	user.server.mapLock.Lock()
	user.server.OnlineMap[user.Name] = user
	user.server.mapLock.Unlock()

	// 广播当前用户上线消息
	user.server.BroadCast(user, "上线啦O(∩_∩)O", "allChat")
	userInfo := UserInfo{
		Name:     user.Name,
		Addr:     user.Addr,
		UserId:   user.userId,
		UserIcon: user.UserIcon,
	}
	user.WriteMessage("loginPara", userInfo)

}

// user.go 脚本

func (user *User) Offline() {

	// 用户下线，将用户从OnlineMap中删除，注意加锁
	user.server.mapLock.Lock()
	delete(user.server.OnlineMap, user.Name)
	user.server.mapLock.Unlock()

	// 广播当前用户下线消息
	user.server.BroadCast(user, "下线了o(╥﹏╥)o", "allChat")
}

// user.go 脚本

func (user *User) DoMessage(buf []byte, length int, redis *redis.Client) {
	//time.Sleep(time.Second * 5)
	//提取用户的消息(去除'\n')
	msg := string(buf[:length-1])
	fmt.Println(msg)
	sep := "&"
	fmt.Println(msg)

	result1, result2, falg := splitStringByFirstChar(msg, sep)
	if !falg {

		fmt.Println("splitStringByFirstChar err:", msg)
		return
	}
	if result1 == "" {
		fmt.Println("result1 is empty:", msg)
		return
	}
	if result2 == "" {
		fmt.Println("result2 is empty:", msg)
		return
	}
	if result1 == "all" {
		user.server.BroadCast(user, msg, "allChat")
		return
	}
	if result1 == "chat" {
		res := strings.Split(result2, "&")
		chatId := res[0]
		chatStr := res[1]
		createTime := time.Now().Unix()
		user.server.mapLock.Lock()
		// 遍历在线用户，找到目标用户
		for _, us := range user.server.OnlineMap {
			flag := redis.HExists(context.Background(), "chatMember_"+chatId, us.userId).Val()
			if flag {
				us.WriteMessage("chatPara", strconv.FormatInt(createTime, 10)+"&"+user.userId+"&"+user.Name+"&"+user.UserIcon+"&"+chatId+"&"+chatStr)
			}
		}
		user.saveChatMsg(redis, chatId, chatStr)
		// 解锁
		user.server.mapLock.Unlock()
		return
	}
	if result1 == "login" {
		ctx := context.Background()
		userId := redis.Get(ctx, "User_Id"+result2).Val()
		if userId == "" {
			// 用户不存在，分配一个新的ID
			userId = redis.Get(ctx, "userIdGenerator").Val()
			if userId == "" {
				userId = "16000000"
			}
			iniId, errs := strconv.ParseInt(userId, 10, 64)
			iniId++
			if errs != nil {
				fmt.Println("strconv.ParseInt err:", errs)
			}
			redis.Set(ctx, "userIdGenerator", strconv.FormatInt(iniId, 10), 0)
			redis.Set(ctx, "User_Id"+result2, userId, 0)

		}
		userName := redis.HGet(ctx, "userName", "userInfo_"+userId).Val()
		if userName == "" {
			// 用户不存在，分配一个新的昵称
			userName = "用户" + userId
			redis.HSet(ctx, "userName", "userInfo_"+userId, userName)

		}
		userIcon := redis.HGet(ctx, "userIcon", "userInfo_"+userId).Val()
		if userIcon == "" {
			randomNum := rand.Intn(20) + 1
			randomNumStr := strconv.Itoa(randomNum)
			userIcon = "head_" + randomNumStr
			redis.HSet(ctx, "userIcon", "userInfo_"+userId, userIcon)
		}

		SetUserId(user, userId)
		SetUserName(user, userName)
		SetUserIcon(user, userIcon)
		// 用户上线
		user.Online()
		return
	}
	if result1 == "getChatIds" {
		sep := "&"
		start, end, falg := splitStringByFirstChar(result2, sep)
		if !falg {
			fmt.Println("splitStringByFirstChar err:", result2)
			return
		}
		startInt, err := strconv.ParseInt(start, 10, 64)
		if err != nil {
			fmt.Println("strconv.ParseInt err:", err)
			return
		}
		endInt, err := strconv.ParseInt(end, 10, 64)
		if err != nil {
			fmt.Println("strconv.ParseInt err:", err)
			return
		}
		chatIds := user.GetUsertChatIds(redis, startInt, endInt)
		allChatInfo := []ChatRoomInfo{}
		for _, chatId := range chatIds {
			ChatRoomName := redis.HGet(context.Background(), "chatRoomName", chatId).Val()
			ChatRoomIcon := redis.HGet(context.Background(), "chatRoomIcon", chatId).Val()
			if ChatRoomName == "PrivateChat" {
				ids := strings.Split(chatId, "_")
				id := "0"
				if ids[1] == user.userId {
					id = ids[2]
				} else {
					id = ids[1]
				}
				ChatRoomName = redis.HGet(context.Background(), "userName", "userInfo_"+id).Val()
				ChatRoomIcon = redis.HGet(context.Background(), "userIcon", "userInfo_"+id).Val()
			}
			chatInfo := ChatRoomInfo{
				ChatRoomId:      chatId,
				ChatRoomName:    ChatRoomName,
				ChatRoomIcon:    ChatRoomIcon,
				ChatRoomContent: user.GetLastChatMsg(redis, chatId)}
			allChatInfo = append(allChatInfo, chatInfo)
		}

		user.WriteMessage("ChatIdsPara", allChatInfo)
		return
	}
	if result1 == "getChatInfo" {
		res := strings.Split(result2, "&")
		chatId := res[0]
		start := res[1]
		end := res[2]
		startInt, err := strconv.ParseInt(start, 10, 64)
		if err != nil {
			fmt.Println("strconv.ParseInt err:", err)
			return
		}

		endInt, err := strconv.ParseInt(end, 10, 64)
		if err != nil {
			fmt.Println("strconv.ParseInt err:", err)
			return
		}
		chatInfo := redis.LRange(context.Background(), "chatContent_"+chatId, startInt, endInt).Val()
		user.WriteMessage("ChatInfoPara", chatInfo)
		return
	}
	if result1 == "createChat" {
		res := strings.Split(result2, "&")
		chatId := res[0]
		chatRoomName := res[1]
		chatRoomIcon := res[2]
		createString := res[3]
		ids := strings.Split(chatId, "_")
		length := len(ids)

		user.CreateChatRoom(redis, chatId, chatRoomName, chatRoomIcon, createString)
		if length > 1 {
			for _, id := range ids {
				AddChatRoom(redis, chatId, id)
				for _, us := range user.server.OnlineMap {
					if us.userId == id {
						ChatRoomName := redis.HGet(context.Background(), "chatRoomName", chatId).Val()
						if ChatRoomName == "PrivateChat" {
							ids := strings.Split(chatId, "_")
							id := "0"
							if ids[1] == user.userId {
								id = ids[2]
							} else {
								id = ids[1]
							}
							ChatRoomName = redis.HGet(context.Background(), "userName", "userInfo_"+id).Val()
							ChatRoomIcon := redis.HGet(context.Background(), "userIcon", "userInfo_"+id).Val()
							chatInfo := ChatRoomInfo{
								ChatRoomId:      chatId,
								ChatRoomName:    ChatRoomName,
								ChatRoomIcon:    ChatRoomIcon,
								ChatRoomContent: user.GetLastChatMsg(redis, chatId)}
							allChatInfo := []ChatRoomInfo{}
							allChatInfo = append(allChatInfo, chatInfo)
							us.WriteMessage("ChatIdsPara", allChatInfo)
						}

					}
				}
			}
		}
		return
	}
}
func (user *User) WriteMessage(method string, msg interface{}) {
	messageInfo := map[string]interface{}{
		"method": method,
		"msg":    msg,
	}
	jsonData, err := json.Marshal(messageInfo)
	if err != nil {
		fmt.Println("json.Marshal err:", err)
		return
	}
	user.Channel <- string(jsonData)
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

func (user *User) GetUsertChatIds(redis *redis.Client, start int64, end int64) []string {

	list := redis.LRange(context.Background(), user.userId+"_chatIds", start, end).Val()
	if len(list) == 0 {
		AddChatRoom(redis, "chatRoomId_1", user.userId)
		list = redis.LRange(context.Background(), user.userId+"_chatIds", 0, 0).Val()
	}
	return list
}

func AddChatRoom(redis *redis.Client, chatId string, userId string) {
	redis.LPush(context.Background(), userId+"_chatIds", chatId)
	redis.HSet(context.Background(), "chatMember_"+chatId, userId, userId)
}

func (user *User) GetLastChatMsg(redis *redis.Client, chatId string) string {
	chatStr := redis.LIndex(context.Background(), "chatContent_"+chatId, 0).Val()
	if chatStr == "" {
		str := user.Name + " 创建了聊天室"
		user.CreateChatRoom(redis, chatId, "公用聊天室", "head_publicRoom", str)
		chatStr = redis.LIndex(context.Background(), "chatContent_"+chatId, 0).Val()
	}
	return chatStr
}
func (user *User) CreateChatRoom(redis *redis.Client, chatId string, chatRoomName string, chatRoomIcon string, createString string) string {
	redis.HSet(context.Background(), "chatRoomName", chatId, chatRoomName)
	redis.HSet(context.Background(), "chatRoomIcon", chatId, chatRoomIcon)
	user.saveChatMsg(redis, chatId, createString)

	return chatId
}

func (user *User) saveChatMsg(redis *redis.Client, chatId string, chatStr string) {
	createTime := time.Now().Unix()
	chatStr = strconv.FormatInt(createTime, 10) + "&" + user.userId + "&" + user.Name + "&" + user.UserIcon + "&" + chatId + "&" + chatStr
	redis.LPush(context.Background(), "chatContent_"+chatId, chatStr)
}
