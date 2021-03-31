package common

import (
	"fmt"
	"testing"
)

func TestQuickSort(t *testing.T) {
	node := GetShuffledDeck()
	fmt.Println(node)
	node = QuickSort(node)
	fmt.Println(node)
}

func BenchmarkNewNode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewNode(uint8(i))
	}
}

func BenchmarkGetShuffledDeck(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetShuffledDeck()
	}
}

func BenchmarkQuickSort(b *testing.B) {
	node := GetShuffledDeck()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = QuickSort(node)
	}
}
