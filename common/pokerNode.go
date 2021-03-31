package common

import "math/rand"

// 0~53
// A234567890JQK 52是大王 53是小王
type PokerVal uint8

type PokerNode struct {
	Val  PokerVal
	Next *PokerNode
}

func NewNode(vals ...uint8) *PokerNode {
	head := PokerNode{}
	node := &head
	for _, u := range vals {
		node.Next = &PokerNode{Val: PokerVal(u)}
		node = node.Next
	}
	return head.Next
}

// 取得一个洗过的牌组
func GetShuffledDeck() *PokerNode {
	shuffle := func(array []uint8) {
		l := len(array)
		for i := l - 1; i >= 0; i-- {
			if i > 0 {
				r := rand.Intn(i)
				array[i], array[r] = array[r], array[i]
			}
		}
	}
	deckArray := make([]uint8, 54)
	for i := range deckArray {
		deckArray[i] = uint8(i + 1)
	}
	shuffle(deckArray)
	return NewNode(deckArray...)
}

func (n PokerNode) Sort() {

}
