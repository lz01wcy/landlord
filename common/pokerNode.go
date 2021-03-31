package common

import (
	"math/rand"
	"strconv"
	"strings"
)

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
		deckArray[i] = uint8(i)
	}
	shuffle(deckArray)
	return NewNode(deckArray...)
}

// 排序
//func (n PokerNode) Sort() {
//
//}

// 快排
func QuickSort(head *PokerNode) *PokerNode {
	if head == nil || head.Next == nil {
		return head
	}
	var left = new(PokerNode)
	var right = new(PokerNode)
	var p1 = left
	var p2 = right
	var p = head.Next
	var base = head // 选取头节点为基准节点
	base.Next = nil
	// 剩余节点中比基准值小就放left里,否则放right里,按照大小拆分为两条链表
	for p != nil {
		pn := p.Next
		p.Next = nil
		if p.Val < base.Val {
			p1.Next = p
			p1 = p1.Next
		} else {
			p2.Next = p
			p2 = p2.Next
		}
		p = pn
	}
	// 递归对两条链表进行排序
	left.Next = QuickSort(left.Next)
	right.Next = QuickSort(right.Next)
	// 先把又链表拼到base后面
	base.Next = right.Next
	// 左链表+基准节点+右链表拼接,左链表有可能是空,所以需要特殊处理下
	if left.Next != nil {
		p = left.Next
		// 找到左链表的最后一个节点
		for p.Next != nil {
			p = p.Next
		}
		// 把base拼接到左链表的末尾
		p.Next = base
		return left.Next
	} else {
		return base
	}
}

// 转换成为[]int
func (receiver *PokerNode) ToIntArray() (res []int) {
	for receiver != nil {
		res = append(res, int(receiver.Val))
		receiver = receiver.Next
	}
	return
}

func (receiver *PokerNode) String() string {
	var sb strings.Builder
	for receiver != nil {
		if sb.Len() != 0 {
			sb.WriteString("->")
		}
		sb.WriteString(strconv.Itoa(int(receiver.Val)))
		receiver = receiver.Next
	}
	return sb.String()
}
