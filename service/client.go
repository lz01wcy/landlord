package service

import (
	"bytes"
	"encoding/json"
	"github.com/astaxie/beego/logs"
	"github.com/gorilla/websocket"
	"landlord/common"
	"net/http"
	"strconv"
	"time"
)

const (
	writeWait      = 1 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512

	// 农民
	RoleFarmer = 0
	// 地主
	RoleLandlord = 1
)

var (
	newline  = []byte{'\n'}
	space    = []byte{' '}
	upGrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	} //不验证origin
)

type UserId int

// 用户信息
type UserInfo struct {
	UserId   UserId `json:"user_id"`
	Username string `json:"username"`
	Coin     int    `json:"coin"`
	Role     int
}

// 客户端
type Client struct {
	conn *websocket.Conn
	// 用户信息
	UserInfo *UserInfo
	// 房子
	Room *Room
	// 牌桌
	Table *Table
	// 手牌
	HandPokers []common.PokerValInt
	// 已准备
	Ready    bool
	IsCalled bool //是否叫完分
	// next 在牌桌上查找下一个客户端
	Next *Client //链表
	// 是机器人
	IsRobot  bool
	toRobot  chan []interface{} //发送给robot的消息
	toServer chan []interface{} //robot发送给服务器
}

//重置状态
func (c *Client) reset() {
	c.UserInfo.Role = 1
	c.HandPokers = make([]common.PokerValInt, 0, 21)
	c.Ready = false
	c.IsCalled = false
}

//发送房间内已有的牌桌信息
func (c *Client) sendRoomTables() {
	res := make([][2]int, 0)
	for _, table := range c.Room.Tables {
		if len(table.TableClients) < 3 {
			res = append(res, [2]int{int(table.TableId), len(table.TableClients)})
		}
	}
	c.sendMsg([]interface{}{common.ResTableList, res})
}

// 发送消息
func (c *Client) sendMsg(msg []interface{}) {
	// 如果是机器人 传入相应通道
	if c.IsRobot {
		c.toRobot <- msg
		return
	}
	// 消息序列化为json
	msgByte, err := json.Marshal(msg)
	if err != nil {
		logs.Error("send msg [%v] marsha1 err:%v", string(msgByte), err)
		return
	}
	// 设置超时时间?
	err = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err != nil {
		logs.Error("send msg SetWriteDeadline [%v] err:%v", string(msgByte), err)
		return
	}
	// 获得一个Writer?
	w, err := c.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		err = c.conn.Close()
		if err != nil {
			logs.Error("close client err: %v", err)
		}
	}
	// 将json写入 即传给客户端
	_, err = w.Write(msgByte)
	if err != nil {
		logs.Error("Write msg [%v] err: %v", string(msgByte), err)
	}
	// 然后关闭Writer
	if err := w.Close(); err != nil {
		err = c.conn.Close()
		if err != nil {
			logs.Error("close err: %v", err)
		}
	}
}

//关闭客户端
func (c *Client) close() {
	if c.Table != nil {
		for _, client := range c.Table.TableClients {
			if c.Table.Creator == c && c != client {
				c.Table.Creator = client
			}
			if c == client.Next {
				client.Next = nil
			}
		}
		if len(c.Table.TableClients) != 1 {
			for _, client := range c.Table.TableClients {
				if client != client.Table.Creator {
					client.Table.Creator.Next = client
				}
			}
		}
		if len(c.Table.TableClients) == 1 {
			c.Table.Creator = nil
			delete(c.Room.Tables, c.Table.TableId)
			return
		}
		delete(c.Table.TableClients, c.UserInfo.UserId)
		if c.Table.State == GamePlaying {
			c.Table.syncUser()
			//c.Table.reset()
		}
		if c.IsRobot {
			close(c.toRobot)
			close(c.toServer)
		}
	}
}

// 泵 处理消息的?
//可能是因为版本问题，导致有些未处理的error
func (c *Client) readPump() {
	// defer
	defer func() {
		//logs.Debug("readPump exit")
		// 关闭conn
		c.conn.Close()
		// 关闭客户端
		c.close()
		// 如果房间允许机器人
		if c.Room.AllowRobot {
			// table不空
			if c.Table != nil {
				// 关闭所有client 不科学吧
				for _, client := range c.Table.TableClients {
					client.close()
				}
			}
		}
	}()
	// websocket的一些设置
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	// 无限循环
	for {
		// 读取消息
		_, message, err := c.conn.ReadMessage()
		// 有报错则日志输出 退出
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logs.Error("websocket user_id[%d] unexpected close error: %v", c.UserInfo.UserId, err)
			}
			return
		}
		// 消息中的换行替换为空格 去除首尾空白符号
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		var data []interface{}
		// 反序列化json
		err = json.Unmarshal(message, &data)
		if err != nil {
			logs.Error("message unmarsha1 err, user_id[%d] err:%v", c.UserInfo.UserId, err)
		} else {
			// 处理请求
			c.wsRequest(data)
		}
	}
}

//心跳
func (c *Client) ping() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
	}()
	for {
		select {
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// http处理
func ServeWs(w http.ResponseWriter, r *http.Request) {
	// 从http升级为websocket?
	conn, err := upGrader.Upgrade(w, r, nil)
	if err != nil {
		logs.Error("upgrader err:%v", err)
		return
	}
	// 根据conn新建客户端
	client := &Client{conn: conn, HandPokers: make([]common.PokerValInt, 0, 21), UserInfo: &UserInfo{}}
	// 取得userId和username
	var userId int
	var username string
	cookie, err := r.Cookie("userid")

	if err != nil {
		logs.Error("get cookie err: %v", err)
	} else {
		userIdStr := cookie.Value
		userId, err = strconv.Atoi(userIdStr)
	}
	cookie, err = r.Cookie("username")

	if err != nil {
		logs.Error("get cookie err: %v", err)
	} else {
		username = cookie.Value
	}

	// 如果userId和username的值有意义
	if userId != 0 && username != "" {
		// 设置client.UserInfo
		client.UserInfo.UserId = UserId(userId)
		client.UserInfo.Username = username
		// 这是啥协程?
		go client.readPump()
		// 启动心跳的协程
		go client.ping()
		return
	}
	logs.Error("user need login first")
	_ = client.conn.Close()
}
