package main

import "strconv"

func encode(value Value) []byte {
	if value.typ == '+' {
		return []byte("+" + value.str + "\r\n")
	}
	if value.typ == '$' && value.str == "" {
		return []byte("$-1\r\n")
	}
	if value.typ == '$' {
		return []byte("$" + strconv.Itoa(len(value.str)) + "\r\n" + value.str + "\r\n")
	}
	if value.typ == '-' {
		return []byte("-" + value.str + "\r\n")
	}
	return nil
}
