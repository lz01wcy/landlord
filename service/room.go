package service

import (
	"github.com/astaxie/beego/logs"
	"sync"
)

var (
	roomManager = RoomManager{
		Rooms: map[int]*Room{
			1: {
				RoomId:      1,
				AllowRobot:  true,
				EntranceFee: 200,
				Tables:      make(map[TableId]*Table),
			},
			2: {
				RoomId:      2,
				AllowRobot:  false,
				EntranceFee: 200,
				Tables:      make(map[TableId]*Table),
			},
		},
	}
)

type RoomId int

// 房间管理器
type RoomManager struct {
	// 读写锁
	Lock sync.RWMutex
	// 房间列表
	Rooms map[int]*Room
	// tableId?
	TableIdInc TableId
}

// 房间
type Room struct {
	// 房间id
	RoomId RoomId
	//读写锁
	Lock sync.RWMutex
	// 允许机器人
	AllowRobot bool
	// 桌子列表
	Tables map[TableId]*Table
	// 入场费
	EntranceFee int
}

//新建牌桌
func (r *Room) newTable(client *Client) (table *Table) {
	// roomManager上锁
	roomManager.Lock.Lock()
	defer roomManager.Lock.Unlock()
	// room上锁
	r.Lock.Lock()
	defer r.Lock.Unlock()
	// 自增 是TableIdInc的当前值?
	roomManager.TableIdInc++
	// 新建table
	table = &Table{
		TableId: roomManager.TableIdInc,
		Creator: client,
		// 初始化字典
		TableClients: make(map[UserId]*Client, 3),
		// 新建GameManager
		GameManager: &GameManager{
			FirstCallScore: client,
			Multiple:       1,
			// 初始化上次出过的牌
			LastShotPoker: make([]int, 0),
			// 初始化牌堆
			Pokers: make([]int, 0, 54),
		},
	}
	// 字典添加table
	r.Tables[table.TableId] = table
	logs.Debug("create new table ok! allow robot :%v", r.AllowRobot)
	return
}

//func init()  {
//	go func() {		//压测
//		time.Sleep(time.Second * 3)
//		for i:=0;i<1;i++{
//			client := &Client{
//				Room:       roomManager.Rooms[1],
//				HandPokers: make([]int, 0, 21),
//				UserInfo: &UserInfo{
//					UserId:   UserId(rand.Intn(10000)),
//					Username: "ROBOT-0",
//					Coin:     10000,
//				},
//				IsRobot:  true,
//				toRobot: make(chan []interface{}, 3),
//				toServer: make(chan []interface{}, 3),
//			}
//			go client.runRobot()
//			table := client.Room.newTable(client)
//			table.joinTable(client)
//		}
//	}()
//}
