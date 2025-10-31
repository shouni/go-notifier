package util

import (
	"github.com/shouni/go-utils/text"
)

// CleanStringFromEmojis は、文字列から絵文字を削除し、Backlogなどのシステムでの文字列処理エラーや表示崩れを防止します。
func CleanStringFromEmojis(s string) string {
	return text.CleanStringFromEmojis(s)
}
