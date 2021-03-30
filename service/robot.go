package service

import (
	"github.com/astaxie/beego/logs"
	"landlord/common"
	"time"
)

// 运行机器人
func (c *Client) runRobot() {
	for {
		select {
		case msg, ok := <-c.toServer:
			if !ok {
				return
			}
			c.wsRequest(msg)
		case msg, ok := <-c.toRobot:
			if !ok {
				return
			}
			logs.Debug("robot [%v] receive  message %v ", c.UserInfo.Username, msg)
			if len(msg) < 1 {
				logs.Error("send to robot [%v],message err ,%v", c.UserInfo.Username, msg)
				return
			}
			if act, ok := msg[0].(int); ok {
				protocolCode := act
				switch protocolCode {
				// 需要发牌
				case common.ResDealPoker:
					time.Sleep(time.Second)
					c.Table.Lock.RLock()
					// 如果自己是第一个叫分 字典叫分
					if c.Table.GameManager.FirstCallScore == c {
						c.autoCallScore()
					}
					c.Table.Lock.RUnlock()
				// 需要叫分
				case common.ResCallScore:
					if len(msg) < 4 {
						logs.Error("ResCallScore msg err:%v", msg)
						return
					}
					time.Sleep(time.Second)
					c.Table.Lock.RLock()
					// 轮到自己且自己没叫分
					if c.Table.GameManager.Turn == c && !c.IsCalled {
						var callEnd bool
						logs.Debug("ResCallScore %t", msg[3])
						if res, ok := msg[3].(bool); ok {
							callEnd = res
						}
						if !callEnd {
							c.autoCallScore()
						}
					}
					c.Table.Lock.RUnlock()
				// 需要出牌
				case common.ResShotPoker:
					time.Sleep(time.Second)
					c.Table.Lock.RLock()
					if c.Table.GameManager.Turn == c {
						c.autoShotPoker()
					}
					c.Table.Lock.RUnlock()
				// 展示牌
				case common.ResShowPoker:
					time.Sleep(time.Second)
					//logs.Debug("robot [%v] role [%v] receive message ResShowPoker turn :%v", c.UserInfo.Username, c.UserInfo.Role, c.Table.GameManager.Turn.UserInfo.Username)
					c.Table.Lock.RLock()
					if c.Table.GameManager.Turn == c || (c.Table.GameManager.Turn == nil && c.UserInfo.Role == RoleLandlord) {
						c.autoShotPoker()
					}
					c.Table.Lock.RUnlock()
					// 游戏结束
				case common.ResGameOver:
					c.Ready = true
				}
			}
		}
	}
}

//自动出牌
func (c *Client) autoShotPoker() {
	//因为机器人休眠一秒后才出牌，有可能因用户退出而关闭chan
	defer func() {
		err := recover()
		if err != nil {
			logs.Warn("autoShotPoker err : %v", err)
		}
	}()
	logs.Debug("robot [%v] auto-shot poker", c.UserInfo.Username)
	shotPokers := make([]int, 0)
	if len(c.Table.GameManager.LastShotPoker) == 0 || c.Table.GameManager.LastShotClient == c {
		shotPokers = append(shotPokers, c.HandPokers[0])
	} else {
		shotPokers = common.CardsAbove(c.HandPokers, c.Table.GameManager.LastShotPoker)
	}
	float64Pokers := make([]interface{}, 0)
	for _, poker := range shotPokers {
		float64Pokers = append(float64Pokers, float64(poker))
	}
	req := []interface{}{float64(common.ReqShotPoker)}
	req = append(req, float64Pokers)
	logs.Debug("robot [%v] autoShotPoker %v", c.UserInfo.Username, float64Pokers)
	c.toServer <- req
}

//自动叫分
func (c *Client) autoCallScore() {
	defer func() {
		err := recover()
		if err != nil {
			logs.Warn("autoCallScore err : %v", err)
		}
	}()
	logs.Debug("robot [%v] autoCallScore", c.UserInfo.Username)
	c.toServer <- []interface{}{float64(common.ReqCallScore), float64(3)}
}
