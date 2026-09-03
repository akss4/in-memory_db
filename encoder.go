package main

import "strconv"

func encode(value Value) []byte {
	if value.typ == '+' { // handles simple string
		return []byte("+" + value.str + "\r\n")
	}
	if value.typ == '$' && value.str == "" { // handles null bulk string
		return []byte("$-1\r\n")
	}
	if value.typ == '$' { // handles bulk string
		return []byte("$" + strconv.Itoa(len(value.str)) + "\r\n" + value.str + "\r\n")
	}
	if value.typ == '-' { // handles error value
		return []byte("-" + value.str + "\r\n")
	}
	if value.typ == ':' { // handles integer value
		return []byte(":" + strconv.Itoa(value.num) + "\r\n")
	}
	encoded := make([]byte, 0) // handles array value
	for _, v := range value.array {
		encoded = append(encoded, encode(v)...)
	}
	header := []byte("*" + strconv.Itoa(len(value.array)) + "\r\n") // header for array value
	return append(header, encoded...)
}
