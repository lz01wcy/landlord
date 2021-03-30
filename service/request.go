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
	case common.ReqCallScore:
		logs.Debug("[%v] ReqCallScore %v", c.UserInfo.Username, data)
		c.Table.Lock.Lock()
		defer c.Table.Lock.Unlock()

		if c.Table.State != GameCallScore {
			logs.Debug("game call score at run time ,%v", c.Table.State)
			return
		}
		if c.Table.GameManager.Turn == c || c.Table.GameManager.FirstCallScore == c {
			c.Table.GameManager.Turn = c.Next
		} else {
			logs.Debug("user [%v] call score turn err ")
		}
		if len(data) < 2 {
			return
		}
		var score int
		if s, ok := data[1].(float64); ok {
			score = int(s)
		}

		if score > 0 && score < c.Table.GameManager.MaxCallScore || score > 3 {
			logs.Error("player[%d] call score[%d] cheat", c.UserInfo.UserId, score)
			return
		}
		if score > c.Table.GameManager.MaxCallScore {
			c.Table.GameManager.MaxCallScore = score
			c.Table.GameManager.MaxCallScoreTurn = c
		}
		c.IsCalled = true
		callEnd := score == 3 || c.Table.allCalled()
		userCall := []interface{}{common.ResCallScore, c.UserInfo.UserId, score, callEnd}
		for _, c := range c.Table.TableClients {
			c.sendMsg(userCall)
		}
		if callEnd {
			logs.Debug("call score end")
			c.Table.callEnd()
		}
	case common.ReqShotPoker:
		logs.Debug("user [%v] ReqShotPoker %v", c.UserInfo.Username, data)
		c.Table.Lock.Lock()
		defer func() {
			c.Table.GameManager.Turn = c.Next
			c.Table.Lock.Unlock()
		}()

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

		if c.Table.State == GameEnd {
			c.Ready = true
			for _, c := range c.Table.TableClients {
				if c.Ready == false {
					return
				}
			}
			logs.Debug("restart")
			c.Table.reset()
		}
	}
}
