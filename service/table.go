package service

import (
	"fmt"
	"github.com/astaxie/beego/logs"
	"landlord/common"
	"math/rand"
	"sort"
	"sync"
	"time"
)

type TableId int

// 状态?
const (
	// 等待玩家
	GameWaitting = iota
	// 叫分状态
	GameCallScore
	// 游戏中
	GamePlaying
	// 游戏结束
	GameEnd
)

// 牌桌
type Table struct {
	// 读写锁
	Lock sync.RWMutex
	// 牌桌id
	TableId TableId
	// 状态
	State int
	// 创建者
	Creator *Client
	// 所有客户端的字典
	TableClients map[UserId]*Client
	// 游戏管理器
	GameManager *GameManager
}

//游戏管理器
type GameManager struct {
	// 当前出牌者?
	Turn *Client
	// 第一个叫分者
	FirstCallScore   *Client //每局轮转
	MaxCallScore     int     //最大叫分
	MaxCallScoreTurn *Client
	// 上次出牌的玩家
	LastShotClient *Client
	// 牌组
	Pokers []int
	// 上次打出的牌
	LastShotPoker []int
	// 加倍倍数
	Multiple int //加倍
}

// 是否所有玩家都已经叫分
func (table *Table) allCalled() bool {
	for _, client := range table.TableClients {
		if !client.IsCalled {
			return false
		}
	}
	return true
}

// 一局结束
func (table *Table) gameOver(client *Client) {
	// 先修改状态
	table.State = GameEnd
	// 结算分数变化
	coin := table.Creator.Room.EntranceFee * table.GameManager.MaxCallScore * table.GameManager.Multiple
	for _, c := range table.TableClients {
		// 消息类型,
		res := []interface{}{common.ResGameOver, client.UserInfo.UserId}
		if c == client {
			res = append(res, coin*2-100)
		} else {
			res = append(res, coin)
		}
		// 每个人的手牌
		for _, cc := range table.TableClients {
			if cc != c {
				userPokers := make([]int, 0, len(cc.HandPokers)+1)
				userPokers = append(userPokers, int(cc.UserInfo.UserId))
				userPokers = append(userPokers, cc.HandPokers...)
				res = append(res, userPokers)
			}
		}
		// 向客户端发出消息
		c.sendMsg(res)
	}
	logs.Debug("table[%d] game over", table.TableId)
}

// 叫分阶段结束
func (table *Table) callEnd() {
	// 状态变化
	table.State = GamePlaying
	// FirstCallScore轮转
	table.GameManager.FirstCallScore = table.GameManager.FirstCallScore.Next
	// 初始化? 是不是分开比较好
	if table.GameManager.MaxCallScoreTurn == nil || table.GameManager.MaxCallScore == 0 {
		table.GameManager.MaxCallScoreTurn = table.Creator
		table.GameManager.MaxCallScore = 1
		//return
	}
	// 地主确定
	landLord := table.GameManager.MaxCallScoreTurn
	// 设置值 农民不设置吗
	landLord.UserInfo.Role = RoleLandlord
	// 出牌者设为地主
	table.GameManager.Turn = landLord
	// 底牌加入地主手牌 直接全加入不好吗 没必要循环吧
	//for _, poker := range table.GameManager.Pokers {
	//	landLord.HandPokers = append(landLord.HandPokers, poker)
	//}
	landLord.HandPokers = append(landLord.HandPokers, table.GameManager.Pokers...)
	// 底牌亮出
	res := []interface{}{common.ResShowPoker, landLord.UserInfo.UserId, table.GameManager.Pokers}
	// 消息发送
	for _, c := range table.TableClients {
		c.sendMsg(res)
	}
}

//客户端加入牌桌
func (table *Table) joinTable(c *Client) {
	// 上锁
	table.Lock.Lock()
	defer table.Lock.Unlock()
	// 人满则拒绝
	if len(table.TableClients) > 2 {
		logs.Error("Player[%d] JOIN Table[%d] FULL", c.UserInfo.UserId, table.TableId)
		return
	}
	logs.Debug("[%v] user [%v] request join table", c.UserInfo.UserId, c.UserInfo.Username)
	// 查看相应id是否存在
	if _, ok := table.TableClients[c.UserInfo.UserId]; ok {
		logs.Error("[%v] user [%v] already in this table", c.UserInfo.UserId, c.UserInfo.Username)
		return
	}
	// 加入
	c.Table = table
	c.Ready = true
	for _, client := range table.TableClients {
		// 找到最后一个client(唯一没有next的) 作为他的next
		if client.Next == nil {
			client.Next = c
			break
		}
	}
	// 字典中加入自己
	table.TableClients[c.UserInfo.UserId] = c
	table.syncUser()
	// 人数=3
	if len(table.TableClients) == 3 {
		c.Next = table.Creator
		table.State = GameCallScore
		table.dealPoker()
	} else if c.Room.AllowRobot {
		go table.addRobot(c.Room)
		logs.Debug("robot join ok")
	}
}

//加入机器人
func (table *Table) addRobot(room *Room) {
	logs.Debug("robot [%v] join table", fmt.Sprintf("ROBOT-%d", len(table.TableClients)))
	// 人数少于3
	if len(table.TableClients) < 3 {
		// 生成机器人
		robot := &Client{
			Room:       room,
			HandPokers: make([]int, 0, 21),
			UserInfo: &UserInfo{
				UserId:   table.getRobotID(),
				Username: fmt.Sprintf("ROBOT-%d", len(table.TableClients)),
				Coin:     10000,
			},
			IsRobot:  true,
			toRobot:  make(chan []interface{}, 3),
			toServer: make(chan []interface{}, 3),
		}
		// 运行协程
		go robot.runRobot()
		// 机器人加入
		table.joinTable(robot)
	}
}

//生成随机robotID
func (table *Table) getRobotID() (robot UserId) {
	time.Sleep(time.Microsecond * 10)
	rand.Seed(time.Now().UnixNano())
	robot = UserId(rand.Intn(10000))
	table.Lock.RLock()
	defer table.Lock.RUnlock()
	if _, ok := table.TableClients[robot]; ok {
		return table.getRobotID()
	}
	return
}

//发牌
func (table *Table) dealPoker() {
	logs.Debug("deal poker")
	// 生成一副牌
	table.GameManager.Pokers = make([]int, 0, 54)
	for i := 0; i < 54; i++ {
		table.GameManager.Pokers = append(table.GameManager.Pokers, i)
	}
	// 洗牌
	table.ShufflePokers()
	// 发牌
	for i := 0; i < 17; i++ {
		for _, client := range table.TableClients {
			client.HandPokers = append(client.HandPokers, table.GameManager.Pokers[len(table.GameManager.Pokers)-1])
			table.GameManager.Pokers = table.GameManager.Pokers[:len(table.GameManager.Pokers)-1]
		}
	}
	// 消息体
	response := make([]interface{}, 0, 3)
	response = append(response, common.ResDealPoker, table.GameManager.FirstCallScore.UserInfo.UserId, nil)
	// 每个客户端获得自己的手牌消息
	for _, client := range table.TableClients {
		// 排序手牌
		sort.Ints(client.HandPokers)
		// 将手牌信息加入 发送
		response[len(response)-1] = client.HandPokers
		client.sendMsg(response)
	}
}

// 聊天
func (table *Table) chat(client *Client, msg string) {
	res := []interface{}{common.ResChat, client.UserInfo.UserId, msg}
	for _, c := range table.TableClients {
		c.sendMsg(res)
	}
}

// 重设
func (table *Table) reset() {
	// 重设GameManage
	table.GameManager = &GameManager{
		FirstCallScore:   table.GameManager.FirstCallScore,
		Turn:             nil,
		MaxCallScore:     0,
		MaxCallScoreTurn: nil,
		LastShotClient:   nil,
		Pokers:           table.GameManager.Pokers[:0],
		LastShotPoker:    table.GameManager.LastShotPoker[:0],
		Multiple:         1,
	}

	// 开房者/房主
	if table.Creator != nil {
		table.Creator.sendMsg([]interface{}{common.ResRestart})
	}
	// 所有客户端重设
	for _, c := range table.TableClients {
		c.reset()
	}
	// 重新发牌 其他东西没问题吗 感觉好多东西都不对啊
	if len(table.TableClients) == 3 {
		// 重设游戏状态
		table.State = GameCallScore
		table.dealPoker()
	}
}

//洗牌
func (table *Table) ShufflePokers() {
	logs.Debug("ShufflePokers")
	r := rand.New(rand.NewSource(time.Now().Unix()))
	i := len(table.GameManager.Pokers)
	// 经典洗牌方法
	for i > 0 {
		randIndex := r.Intn(i)
		table.GameManager.Pokers[i-1], table.GameManager.Pokers[randIndex] = table.GameManager.Pokers[randIndex], table.GameManager.Pokers[i-1]
		i--
	}
}

//同步用户信息
func (table *Table) syncUser() {
	logs.Debug("sync user")
	// 消息体
	response := make([]interface{}, 0, 3)
	response = append(response, common.ResJoinTable, table.TableId)
	// 不应该是3个用户吗
	//tableUsers := make([][2]interface{}, 0, 2)
	tableUsers := make([][2]interface{}, 0, 3)
	// 房主
	current := table.Creator
	// 不考虑同步时玩家人数不足 current可能为nil吗?
	//for i := 0; i < len(table.TableClients); i++ {
	//	tableUsers = append(tableUsers, [2]interface{}{current.UserInfo.UserId, current.UserInfo.Username})
	//	current = current.Next
	//}
	for current != nil {
		tableUsers = append(tableUsers, [2]interface{}{current.UserInfo.UserId, current.UserInfo.Username})
		current = current.Next
	}
	response = append(response, tableUsers)
	for _, client := range table.TableClients {
		// 向每个客户端发送消息
		client.sendMsg(response)
	}
}
