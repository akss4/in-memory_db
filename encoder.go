package main

func encode(value Value) []byte {
	if value.typ == '+' {
		return []byte("+" + value.str + "\r\n")
	}
	return nil
}
