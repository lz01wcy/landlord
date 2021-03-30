package common

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

var (
	// 牌型的组合
	Pokers = make(map[string]*Combination, 16384)
	// 牌型
	TypeToPokers = make(map[string][]*Combination, 38)
)

// 组合
type Combination struct {
	Type  string
	Score int
	Poker string
}

// 说白了是Pokers和TypeToPokers的生成
// 读json比直接生成快吗
func init() {
	// 读取rule.json
	// 不存在则创建
	path := "./rule.json"
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		write()
	}
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var jsonStrByte []byte
	for {
		buf := make([]byte, 1024)
		readNum, err := file.Read(buf)
		if err != nil && err != io.EOF {
			panic(err)
		}
		for i := 0; i < readNum; i++ {
			jsonStrByte = append(jsonStrByte, buf[i])
		}
		if readNum == 0 {
			break
		}
	}
	var rule = make(map[string][]string)
	err = json.Unmarshal(jsonStrByte, &rule)
	if err != nil {
		fmt.Printf("json unmarsha1 err:%v \n", err)
		return
	}
	for pokerType, pokers := range rule {
		for score, poker := range pokers {
			cards := SortStr(poker)
			p := &Combination{
				Type:  pokerType,
				Score: score,
				Poker: cards,
			}
			Pokers[cards] = p
			TypeToPokers[pokerType] = append(TypeToPokers[pokerType], p)
		}
	}
}
