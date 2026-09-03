package main

import "testing"

func TestParseSimpleString(t *testing.T) { //testing simple strings
	data := []byte("+OK\r\n")

	value, consumed, err := parse(data) //actual parse function from prod

	if err != nil {
		t.Fatalf("expected no error, got %v", err) // i expect no error, else fatal
	}

	if value.typ != '+' {
		t.Fatalf("expected type '+', got %q", value.typ) // i expeted type +, else fatal
	}

	if value.str != "OK" {
		t.Fatalf("expected string 'OK', got %q", value.str) // i expected string OK, else fatal
	}

	if consumed != 5 {
		t.Fatalf("expected 5 bytes consumed, got %d", consumed) // i expected 5 bytes consumed, else fatal
	}
}

func TestParseInteger(t *testing.T) {
	data := []byte(":123\r\n")

	value, consumed, err := parse(data)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if value.typ != ':' {
		t.Fatalf("expected type ':', got %q", value.typ)
	}

	if value.num != 123 {
		t.Fatalf("expected number 123, got %d", value.num)
	}

	if consumed != 6 {
		t.Fatalf("expected 6 bytes consumed, got %d", consumed)
	}
}

func TestParseBulkString(t *testing.T) {
	data := []byte("$5\r\nhello\r\n")

	value, consumed, err := parse(data)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if value.typ != '$' {
		t.Fatalf("expected type '$', got %q", value.typ)
	}

	if value.str != "hello" {
		t.Fatalf("expected string 'hello', got %q", value.str)
	}

	if consumed != 11 {
		t.Fatalf("expected 11 bytes consumed, got %d", consumed)
	}
}

func TestParseNullBulkString(t *testing.T) {
	data := []byte("$-1\r\n")

	value, consumed, err := parse(data)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if value.typ != '$' {
		t.Fatalf("expected type '$', got %q", value.typ)
	}

	if value.str != "" {
		t.Fatalf("expected empty string, got %q", value.str)
	}

	if consumed != 5 {
		t.Fatalf("expected 5 bytes consumed, got %d", consumed)
	}
}

func TestParseArray(t *testing.T) {
	data := []byte("*2\r\n$4\r\nPING\r\n$4\r\nPONG\r\n")

	value, consumed, err := parse(data)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if value.typ != '*' {
		t.Fatalf("expected type '*', got %q", value.typ)
	}

	if len(value.array) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(value.array))
	}

	if value.array[0].str != "PING" {
		t.Fatalf("expected first element 'PING', got %q", value.array[0].str)
	}

	if value.array[1].str != "PONG" {
		t.Fatalf("expected second element 'PONG', got %q", value.array[1].str)
	}

	if consumed != len(data) {
		t.Fatalf("expected %d bytes consumed, got %d", len(data), consumed)
	}
}
