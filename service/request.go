package service

import (
	"github.com/astaxie/beego/logs"
	"landlord/common"
)

// 处理websocket请求
func (c *Client) wsRequest(data []interface{}) {
	defer func() {
		if r := recover(); r != nil {
			logs.Error("wsRequest panic:%v ", r)
		}
	}()
	// 数据为空 退出
	if len(data) < 1 {
		return
	}
	var req int
	// 将数据第一位断言为整数
	if r, ok := data[0].(float64); ok {
		req = int(r)
	}
	// 分类处理
	switch req {
	case common.ReqCheat:
		if len(data) < 2 {
			logs.Error("user [%d] request ReqCheat ,but missing user id", c.UserInfo.UserId)
			return
		}

	case common.ReqLogin:
		c.sendMsg([]interface{}{common.ResLogin, c.UserInfo.UserId, c.UserInfo.Username})

	case common.ReqRoomList:
		c.sendMsg([]interface{}{common.ResRoomList})

	case common.ReqTableList:
		c.sendRoomTables()

	case common.ReqJoinRoom:
		if len(data) < 2 {
			logs.Error("user [%d] request join room ,but missing room id", c.UserInfo.UserId)
			return
		}
		var roomId int
		if id, ok := data[1].(float64); ok {
			roomId = int(id)
		}
		roomManager.Lock.RLock()
		defer roomManager.Lock.RUnlock()
		if room, ok := roomManager.Rooms[roomId]; ok {
			c.Room = room
			res := make([][2]int, 0)
			for _, table := range c.Room.Tables {
				if len(table.TableClients) < 3 {
					res = append(res, [2]int{int(table.TableId), len(table.TableClients)})
				}
			}
			c.sendMsg([]interface{}{common.ResJoinRoom, res})
		}

	case common.ReqNewTable:
		table := c.Room.newTable(c)
		table.joinTable(c)

	case common.ReqJoinTable:
		if len(data) < 2 {
			return
		}
		var tableId TableId
		if id, ok := data[1].(float64); ok {
			tableId = TableId(id)
		}
		if c.Room == nil {
			return
		}
		c.Room.Lock.RLock()
		defer c.Room.Lock.RUnlock()

		if table, ok := c.Room.Tables[tableId]; ok {
			table.joinTable(c)
		}
		c.sendRoomTables()
	case common.ReqDealPoker:
		if c.Table.State == GameEnd {
			c.Ready = true
		}
		// 叫分开始
	case common.ReqCallScore:
		logs.Debug("[%v] ReqCallScore %v", c.UserInfo.Username, data)
		// 上锁
		c.Table.Lock.Lock()
		defer c.Table.Lock.Unlock()
		// 状态不对应 返回
		if c.Table.State != GameCallScore {
			logs.Debug("game call score at run time ,%v", c.Table.State)
			return
		}
		// 如果 c是地主
		if c.Table.GameManager.Turn == c || c.Table.GameManager.FirstCallScore == c {
			c.Table.GameManager.Turn = c.Next
		} else {
			logs.Debug("user [%v] call score turn err ")
		}
		// 数据小于2 不正常 退出
		if len(data) < 2 {
			return
		}
		// 声明叫分
		var score int
		// 从data中读取score 断言为float64 转换为int
		if s, ok := data[1].(float64); ok {
			score = int(s)
		}
		// 叫分不合理 应该是1,2,3中一个 这个判断小于等于0没处理吧
		//if score > 0 && score < c.Table.GameManager.MaxCallScore || score > 3 {
		if score <= 0 || score <= c.Table.GameManager.MaxCallScore || score > 3 {
			logs.Error("player[%d] call score[%d] cheat", c.UserInfo.UserId, score)
			return
		}
		// 如果比上一个叫分的人更大
		if score > c.Table.GameManager.MaxCallScore {
			c.Table.GameManager.MaxCallScore = score
			c.Table.GameManager.MaxCallScoreTurn = c
		}
		// 该c叫分结束
		c.IsCalled = true
		// 如果叫分到3或者全部人叫完
		callEnd := score == 3 || c.Table.allCalled()
		// 生成一个消息体
		userCall := []interface{}{common.ResCallScore, c.UserInfo.UserId, score, callEnd}
		// 发送消息给每个人
		for _, c := range c.Table.TableClients {
			c.sendMsg(userCall)
		}
		// 如果叫分结束
		if callEnd {
			logs.Debug("call score end")
			c.Table.callEnd()
		}

		// 请求打出牌
	case common.ReqShotPoker:
		logs.Debug("user [%v] ReqShotPoker %v", c.UserInfo.Username, data)
		// 上锁
		c.Table.Lock.Lock()
		// defer:解锁 当前玩家右移
		defer func() {
			c.Table.GameManager.Turn = c.Next
			c.Table.Lock.Unlock()
		}()
		// 如果自己不是当前玩家 退出
		if c.Table.GameManager.Turn != c {
			logs.Error("shot poker err,not your [%d] turn .[%d]", c.UserInfo.UserId, c.Table.GameManager.Turn.UserInfo.UserId)
			return
		}

		if len(data) > 1 {
			if pokers, ok := data[1].([]interface{}); ok {
				shotPokers := make([]int, 0, len(pokers))
				for _, item := range pokers {
					if i, ok := item.(float64); ok {
						poker := int(i)
						inHand := false
						for _, handPoker := range c.HandPokers {
							if handPoker == poker {
								inHand = true
								break
							}
						}
						if inHand {
							shotPokers = append(shotPokers, poker)
						} else {
							logs.Warn("player[%d] play non-exist poker", c.UserInfo.UserId)
							res := []interface{}{common.ResShotPoker, c.UserInfo.UserId, []int{}}
							for _, c := range c.Table.TableClients {
								c.sendMsg(res)
							}
							return
						}
					}
				}
				if len(shotPokers) > 0 {
					compareRes, isMulti := common.ComparePoker(c.Table.GameManager.LastShotPoker, shotPokers)
					if c.Table.GameManager.LastShotClient != c && compareRes < 1 {
						logs.Warn("player[%d] shot poker %v small than last shot poker %v ", c.UserInfo.UserId, shotPokers, c.Table.GameManager.LastShotPoker)
						res := []interface{}{common.ResShotPoker, c.UserInfo.UserId, []int{}}
						for _, c := range c.Table.TableClients {
							c.sendMsg(res)
						}
						return
					}
					if isMulti {
						c.Table.GameManager.Multiple *= 2
					}
					c.Table.GameManager.LastShotClient = c
					c.Table.GameManager.LastShotPoker = shotPokers
					for _, shotPoker := range shotPokers {
						for i, poker := range c.HandPokers {
							if shotPoker == poker {
								copy(c.HandPokers[i:], c.HandPokers[i+1:])
								c.HandPokers = c.HandPokers[:len(c.HandPokers)-1]
								break
							}
						}
					}
				}
				res := []interface{}{common.ResShotPoker, c.UserInfo.UserId, shotPokers}
				for _, c := range c.Table.TableClients {
					c.sendMsg(res)
				}
				if len(c.HandPokers) == 0 {
					c.Table.gameOver(c)
				}
			}
		}

		//case common.ReqGameOver:
	case common.ReqChat:
		if len(data) > 1 {
			switch data[1].(type) {
			case string:
				c.Table.chat(c, data[1].(string))
			}
		}
	case common.ReqRestart:
		c.Table.Lock.Lock()
		defer c.Table.Lock.Unlock()
		// 如果牌桌状态为结束
		if c.Table.State == GameEnd {
			// 客户端准备完成
			c.Ready = true
			// 判断是否全部准备完成
			for _, c := range c.Table.TableClients {
				if c.Ready == false {
					return
				}
			}
			logs.Debug("restart")
			// 重设Table
			c.Table.reset()
		}
	}
}
