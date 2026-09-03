package main

import "testing"

func resetDatabase() {
	store = make(map[string]string)
	hash = make(map[string]map[string]string)
}

func TestSetAndGet(t *testing.T) {
	resetDatabase()
	set := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "SET"},
			{typ: '$', str: "name"},
			{typ: '$', str: "bunny"},
		},
	}

	handleCommand(set)

	get := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "GET"},
			{typ: '$', str: "name"},
		},
	}

	result := handleCommand(get)

	if result.str != "bunny" {
		t.Fatalf("expected bunny, got %q", result.str)
	}
}

func TestHSetAndHGet(t *testing.T) {
	resetDatabase() // testing hset and hget commands
	hset := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "HSET"},
			{typ: '$', str: "user"},
			{typ: '$', str: "name"},
			{typ: '$', str: "bunny"},
		},
	}

	handleCommand(hset)

	hget := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "HGET"},
			{typ: '$', str: "user"},
			{typ: '$', str: "name"},
		},
	}

	result := handleCommand(hget)

	if result.str != "bunny" {
		t.Fatalf("expected bunny, got %q", result.str)
	}
}

func TestHSetAndHGetAll(t *testing.T) {
	resetDatabase() // testing hset and (purpose)hgetall commands
	hsetName := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "HSET"},
			{typ: '$', str: "user"},
			{typ: '$', str: "name"},
			{typ: '$', str: "bunny"},
		},
	}

	handleCommand(hsetName)

	hsetAge := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "HSET"},
			{typ: '$', str: "user"},
			{typ: '$', str: "age"},
			{typ: '$', str: "20"},
		},
	}

	handleCommand(hsetAge)

	hgetall := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "HGETALL"},
			{typ: '$', str: "user"},
		},
	}

	result := handleCommand(hgetall)
	if len(result.array) != 4 {
		t.Fatalf("expected 4 values, got %d", len(result.array))
	}

	got := make(map[string]string)

	for i := 0; i < len(result.array); i += 2 {
		got[result.array[i].str] = result.array[i+1].str
	}

	if got["name"] != "bunny" {
		t.Fatalf("expected name=bunny, got %q", got["name"])
	}

	if got["age"] != "20" {
		t.Fatalf("expected age=20, got %q", got["age"])
	}
}

func TestHSetAndHDel(t *testing.T) {
	resetDatabase() //HSET → HDEL → HGET
	hset := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "HSET"},
			{typ: '$', str: "user"},
			{typ: '$', str: "name"},
			{typ: '$', str: "bunny"},
		},
	}

	handleCommand(hset)

	hdel := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "HDEL"},
			{typ: '$', str: "user"},
			{typ: '$', str: "name"},
		},
	}

	result := handleCommand(hdel)

	if result.num != 1 {
		t.Fatalf("expected 1, got %d", result.num)
	}

	hget := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "HGET"},
			{typ: '$', str: "user"},
			{typ: '$', str: "name"},
		},
	}

	result = handleCommand(hget)

	if result.typ != '$' || result.str != "" {
		t.Fatalf("expected null value, got %+v", result)
	}
}

func TestFlushDB(t *testing.T) {
	resetDatabase()
	set := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "SET"},
			{typ: '$', str: "name"},
			{typ: '$', str: "bunny"},
		},
	}

	handleCommand(set)

	hset := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "HSET"},
			{typ: '$', str: "user"},
			{typ: '$', str: "age"},
			{typ: '$', str: "20"},
		},
	}

	handleCommand(hset)

	flush := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "FLUSHDB"},
		},
	}

	result := handleCommand(flush)

	if result.str != "OK" {
		t.Fatalf("expected OK, got %q", result.str)
	}

	get := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "GET"},
			{typ: '$', str: "name"},
		},
	}

	result = handleCommand(get)

	if result.typ != '$' || result.str != "" {
		t.Fatalf("expected name to be deleted, got %+v", result)
	}

	hget := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "HGET"},
			{typ: '$', str: "user"},
			{typ: '$', str: "age"},
		},
	}

	result = handleCommand(hget)

	if result.typ != '$' || result.str != "" {
		t.Fatalf("expected hash to be deleted, got %+v", result)
	}
}

func TestGetMissingKey(t *testing.T) {
	resetDatabase() // non exisiting key for get command
	get := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "GET"},
			{typ: '$', str: "does-not-exist"},
		},
	}

	result := handleCommand(get)

	if result.typ != '$' || result.str != "" {
		t.Fatalf("expected null value, got %+v", result)
	}
}

func TestSetOverwrite(t *testing.T) {
	resetDatabase()
	set1 := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "SET"},
			{typ: '$', str: "name"},
			{typ: '$', str: "bunny"},
		},
	}

	handleCommand(set1)

	set2 := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "SET"},
			{typ: '$', str: "name"},
			{typ: '$', str: "rabbit"},
		},
	}

	handleCommand(set2)

	get := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "GET"},
			{typ: '$', str: "name"},
		},
	}

	result := handleCommand(get)

	if result.str != "rabbit" {
		t.Fatalf("expected rabbit, got %q", result.str)
	}
}

func TestHSetOverwrite(t *testing.T) {
	resetDatabase()
	hset1 := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "HSET"},
			{typ: '$', str: "user"},
			{typ: '$', str: "name"},
			{typ: '$', str: "bunny"},
		},
	}

	handleCommand(hset1)

	hset2 := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "HSET"},
			{typ: '$', str: "user"},
			{typ: '$', str: "name"},
			{typ: '$', str: "rabbit"},
		},
	}

	handleCommand(hset2)

	hget := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "HGET"},
			{typ: '$', str: "user"},
			{typ: '$', str: "name"},
		},
	}

	result := handleCommand(hget)

	if result.str != "rabbit" {
		t.Fatalf("expected rabbit, got %q", result.str)
	}
}

func TestHDelMissingField(t *testing.T) {
	resetDatabase() // evvenually failed due to shared memory of previous test
	hdel := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "HDEL"},
			{typ: '$', str: "user"},
			{typ: '$', str: "name"},
		},
	}

	result := handleCommand(hdel)

	if result.num != 0 {
		t.Fatalf("expected 0, got %d", result.num)
	}
}

func TestHGetMissingHash(t *testing.T) {
	resetDatabase()
	hget := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "HGET"},
			{typ: '$', str: "user"},
			{typ: '$', str: "name"},
		},
	}

	result := handleCommand(hget)

	if result.typ != '$' || result.str != "" {
		t.Fatalf("expected null value, got %+v", result)
	}
}

func TestHGetMissingField(t *testing.T) {
	resetDatabase()
	hset := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "HSET"},
			{typ: '$', str: "user"},
			{typ: '$', str: "age"},
			{typ: '$', str: "20"},
		},
	}

	handleCommand(hset)

	hget := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "HGET"},
			{typ: '$', str: "user"},
			{typ: '$', str: "name"},
		},
	}

	result := handleCommand(hget)

	if result.typ != '$' || result.str != "" {
		t.Fatalf("expected null value, got %+v", result)
	}
}

func TestSetInvalidArguments(t *testing.T) {
	resetDatabase()

	set := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "SET"},
			{typ: '$', str: "name"},
		},
	}

	result := handleCommand(set)

	if result.typ != '-' {
		t.Fatalf("expected error response, got type %q", result.typ)
	}
}
