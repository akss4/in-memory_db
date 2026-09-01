package main

import (
	"bytes"
	"errors"
	"strconv"
)

var errIncomplete = errors.New("Incomplete Data")

type Value struct {
	typ   byte
	str   string
	num   int
	array []Value
}

func parse(data []byte) (Value, int, error) {
	if len(data) == 0 {
		return Value{}, 0, errors.New("No Data")
	}
	switch data[0] {
	case '+': //Simple string
		end := bytes.Index(data, []byte("\r\n"))
		if end == -1 {
			return Value{}, 0, errIncomplete
		}
		return Value{typ: '+',
			str: string(data[1:end])}, end + 2, nil

	case ':':
		// Integer
		end := bytes.Index(data, []byte("\r\n"))
		if end == -1 {
			return Value{}, 0, errIncomplete
		}
		num, err := strconv.Atoi(string(data[1:end]))
		if err != nil {
			return Value{}, 0, err
		}
		return Value{typ: ':',
			num: num}, end + 2, nil

	case '-': // error value
		end := bytes.Index(data, []byte("\r\n"))
		if end == -1 {
			return Value{}, 0, errIncomplete
		}
		return Value{typ: '-',
			str: string(data[1:end])}, end + 2, nil

	case '$': //// Bulk String: $5\r\nhello\r\n
		// First read the length, then read exactly that many bytes as the string.
		// Also verify that the string is followed by \r\n.
		end := bytes.Index(data, []byte("\r\n"))
		if end == -1 {
			return Value{}, 0,
				errIncomplete
		}
		length, err := strconv.Atoi(string(data[1:end]))
		if err != nil {
			return Value{}, 0, err
		}

		start := end + 2
		dataEnd := start + length
		if dataEnd > len(data) {
			return Value{}, 0,
				errIncomplete
		}
		if dataEnd+2 > len(data) {
			return Value{}, 0,
				errIncomplete
		}
		if data[dataEnd] != '\r' ||
			data[dataEnd+1] != '\n' {
			return Value{}, 0,
				errors.New("Invalid Data")
		}
		return Value{typ: '$',
			str: string(data[start:dataEnd])}, dataEnd + 2, nil

	case '*':
		// Array: *2\r\n:10\r\n:20\r\n
		// First read the number of elements, then recursively parse each element.
		// Store all parsed Values inside []Value.
		end := bytes.Index(data, []byte("\r\n"))
		if end == -1 {
			return Value{}, 0, errIncomplete
		}
		length, err := strconv.Atoi(string(data[1:end]))
		if err != nil {
			return Value{}, 0, err
		}
		values := make([]Value, 0, length)
		pos := end + 2

		for i := 0; i < length; i++ {
			value, consumed, err := parse(data[pos:])
			if err != nil {
				return Value{}, 0, err
			}
			values = append(values, value)
			pos = pos + consumed
		}
		return Value{typ: '*',
			array: values}, pos, nil
	}
	return Value{}, 0, errors.New("Invalid Data")
}
