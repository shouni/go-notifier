package util

import (
	"strings"

	"github.com/forPelevin/gomoji"
)

// cleanStringFromEmojis は、文字列から絵文字を削除し、Backlogなどのシステムでの文字列処理エラーや表示崩れを防止します。
func CleanStringFromEmojis(s string) string {
	// gomoji ライブラリを使用して絵文字を削除
	s = gomoji.RemoveEmojis(s)

	// 不必要な連続する空白文字を一つにまとめる
	s = strings.Join(strings.Fields(s), " ")

	return s
}
