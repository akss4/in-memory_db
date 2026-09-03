package main

import (
	"path/filepath"
	"testing"
)

func TestAofWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "database.aof")

	aof, err := NewAof(path)
	if err != nil {
		t.Fatalf("failed to create AOF: %v", err)
	}
	defer aof.Close()

	value := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "SET"},
			{typ: '$', str: "name"},
			{typ: '$', str: "bunny"},
		},
	}

	err = aof.Write(value)
	if err != nil {
		t.Fatalf("failed to write AOF: %v", err)
	}

	var got Value

	err = aof.Read(func(value Value) {
		got = value
	})
	if err != nil {
		t.Fatalf("failed to read AOF: %v", err)
	}

	if got.typ != '*' {
		t.Fatalf("expected type '*', got %q", got.typ)
	}

	if len(got.array) != 3 {
		t.Fatalf("expected 3 values, got %d", len(got.array))
	}

	if got.array[0].str != "SET" ||
		got.array[1].str != "name" ||
		got.array[2].str != "bunny" {
		t.Fatalf("unexpected value: %+v", got.array)
	}
}

func TestAofMultipleWritesAndRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "database.aof")

	aof, err := NewAof(path)
	if err != nil {
		t.Fatalf("failed to create AOF: %v", err)
	}
	defer aof.Close()

	values := []Value{
		{
			typ: '*',
			array: []Value{
				{typ: '$', str: "SET"},
				{typ: '$', str: "name"},
				{typ: '$', str: "bunny"},
			},
		},
		{
			typ: '*',
			array: []Value{
				{typ: '$', str: "SET"},
				{typ: '$', str: "age"},
				{typ: '$', str: "20"},
			},
		},
		{
			typ: '*',
			array: []Value{
				{typ: '$', str: "HSET"},
				{typ: '$', str: "user"},
				{typ: '$', str: "city"},
				{typ: '$', str: "Delhi"},
			},
		},
	}

	for _, value := range values {
		err := aof.Write(value)
		if err != nil {
			t.Fatalf("failed to write AOF: %v", err)
		}
	}

	var got []Value

	err = aof.Read(func(value Value) {
		got = append(got, value)
	})
	if err != nil {
		t.Fatalf("failed to read AOF: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 values, got %d", len(got))
	}

	if got[0].array[2].str != "bunny" {
		t.Fatalf("expected bunny, got %q", got[0].array[2].str)
	}

	if got[1].array[2].str != "20" {
		t.Fatalf("expected 20, got %q", got[1].array[2].str)
	}

	if got[2].array[3].str != "Delhi" {
		t.Fatalf("expected Delhi, got %q", got[2].array[3].str)
	}
}

func TestAofClear(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "database.aof")

	aof, err := NewAof(path)
	if err != nil {
		t.Fatalf("failed to create AOF: %v", err)
	}
	defer aof.Close()

	value := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "SET"},
			{typ: '$', str: "name"},
			{typ: '$', str: "bunny"},
		},
	}

	err = aof.Write(value)
	if err != nil {
		t.Fatalf("failed to write AOF: %v", err)
	}

	err = aof.Clear()
	if err != nil {
		t.Fatalf("failed to clear AOF: %v", err)
	}

	var got []Value

	err = aof.Read(func(value Value) {
		got = append(got, value)
	})
	if err != nil {
		t.Fatalf("failed to read AOF: %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("expected empty AOF, got %d values", len(got))
	}
}

func TestAofPersistenceAfterReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "database.aof")

	// First AOF instance: write data
	aof, err := NewAof(path)
	if err != nil {
		t.Fatalf("failed to create AOF: %v", err)
	}

	value := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "SET"},
			{typ: '$', str: "name"},
			{typ: '$', str: "bunny"},
		},
	}

	err = aof.Write(value)
	if err != nil {
		t.Fatalf("failed to write AOF: %v", err)
	}

	aof.Close()

	// Second AOF instance: reopen the same file
	aof, err = NewAof(path)
	if err != nil {
		t.Fatalf("failed to reopen AOF: %v", err)
	}
	defer aof.Close()

	var got Value

	err = aof.Read(func(value Value) {
		got = value
	})
	if err != nil {
		t.Fatalf("failed to read AOF: %v", err)
	}

	if got.array[0].str != "SET" ||
		got.array[1].str != "name" ||
		got.array[2].str != "bunny" {
		t.Fatalf("unexpected value after reopen: %+v", got.array)
	}
}

func TestAofReplay(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "database.aof")

	// Write a command to the AOF.
	aof, err := NewAof(path)
	if err != nil {
		t.Fatalf("failed to create AOF: %v", err)
	}

	set := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "SET"},
			{typ: '$', str: "name"},
			{typ: '$', str: "bunny"},
		},
	}

	err = aof.Write(set)
	if err != nil {
		t.Fatalf("failed to write AOF: %v", err)
	}

	aof.Close()

	// Start with a completely empty in-memory database.
	resetDatabase()

	// Reopen the AOF and replay its commands.
	aof, err = NewAof(path)
	if err != nil {
		t.Fatalf("failed to reopen AOF: %v", err)
	}
	defer aof.Close()

	err = aof.Read(func(value Value) {
		handleCommand(value)
	})
	if err != nil {
		t.Fatalf("failed to replay AOF: %v", err)
	}

	// Verify that replay rebuilt the database.
	get := Value{
		typ: '*',
		array: []Value{
			{typ: '$', str: "GET"},
			{typ: '$', str: "name"},
		},
	}

	result := handleCommand(get)

	if result.str != "bunny" {
		t.Fatalf("expected bunny after replay, got %q", result.str)
	}

}
