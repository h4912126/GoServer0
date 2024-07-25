package main

import (
	"strings"
)

func splitStringByFirstChar(s, sep string) (string, string, bool) {
	// 找到分隔符在字符串中首次出现的索引
	index := strings.Index(s, sep)

	// 如果没有找到分隔符，则返回原始字符串和两个空字符串，以及一个表示未找到的布尔值
	if index == -1 {
		return s, "", false
	}

	// 否则，使用索引来分割字符串
	// 注意：字符串切片是左闭右开的，所以index+len(sep)会跳过分隔符
	before := s[:index]
	after := s[index+len(sep):]

	return before, after, true
}
