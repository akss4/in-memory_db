package main

import (
	"strings"
	"sync"
)

var store = make(map[string]string)
var storeMu = sync.RWMutex{} // for basic string commands

var hash = make(map[string]map[string]string)
var hashMu = sync.RWMutex{} // for hash commands

func handleCommand(value Value) Value {
	if value.typ != '*' {
		return Value{}
	}
	if len(value.array) == 0 {
		return Value{}
	}
	if value.array[0].typ != '$' {
		return Value{}
	}
	command := value.array[0].str
	command = strings.ToUpper(command)

	if command == "PING" {
		if len(value.array) > 1 {
			return Value{
				typ: '+',
				str: value.array[1].str,
			}
		}
		return Value{
			typ: '+',
			str: "PONG",
		}
	}
	if command == "SET" {
		if len(value.array) != 3 {
			return Value{
				typ: '-',
				str: "ERR wrong number of argument for 'SET' command",
			}
		}
		key := value.array[1].str
		val := value.array[2].str
		storeMu.Lock()
		store[key] = val
		storeMu.Unlock()
		return Value{
			typ: '+',
			str: "OK",
		}
	}

	if command == "GET" {
		if len(value.array) != 2 {
			return Value{
				typ: '-',
				str: "ERR wrong number of argument for 'GET' command",
			}

		}
		key := value.array[1].str
		storeMu.RLock()
		val, ok := store[key]
		storeMu.RUnlock()
		if !ok {
			return Value{
				typ: '$',
				str: "",
			}
		}
		return Value{
			typ: '$',
			str: val, // get returns the value of the key if it exists, otherwise it returns an empty string
		}
	}

	if command == "HSET" {
		if len(value.array) != 4 {
			return Value{
				typ: '-',
				str: "ERR wrong number of argument for 'HSET' command",
			}
		}
		key := value.array[1].str   // outer map like the index of the hash it has a map in itself too it is a string mapped to a map of string to string  and field andd val are the key and value of the stuff inside the map
		field := value.array[2].str // inner map field
		val := value.array[3].str   // inner map value
		hashMu.Lock()
		if hash[key] == nil {
			hash[key] = make(map[string]string)
		}
		hash[key][field] = val
		hashMu.Unlock()

		return Value{
			typ: '+',
			str: "OK", // hset acceptance of values
		}

	}

	if command == "HGET" {
		if len(value.array) != 3 {
			return Value{
				typ: '-',
				str: "ERR wrong number of argument for 'HGET' command",
			}
		}
		key := value.array[1].str
		field := value.array[2].str
		hashMu.RLock()
		val, ok := hash[key][field]
		hashMu.RUnlock()
		if !ok {
			return Value{
				typ: '$',
				str: "",
			}
		}
		return Value{
			typ: '$',
			str: val,
		}
	}

	if command == "HGETALL" {
		if len(value.array) != 2 {
			return Value{
				typ: '-',
				str: "ERR wrong number of argument for 'HGETALL' command",
			}
		}
		key := value.array[1].str
		hashMu.RLock()
		val := hash[key]
		response := make([]Value, 0, len(val)*2)
		for field, value := range val {
			response = append(response,
				Value{
					typ: '$',
					str: field,
				},
				Value{
					typ: '$',
					str: value,
				},
			)
		}
		hashMu.RUnlock()
		return Value{
			typ:   '*',
			array: response,
		}
	}

	if command == "HDEL" {
		if len(value.array) != 3 {
			return Value{
				typ: '-',
				str: "ERR wrong number of argument for 'HDEL' command",
			}
		}
		key := value.array[1].str
		field := value.array[2].str
		hashMu.Lock()
		_, ok := hash[key][field]
		if !ok {
			hashMu.Unlock()
			return Value{
				typ: ':',
				num: 0,
			}
		}
		delete(hash[key], field)
		hashMu.Unlock()
		return Value{
			typ: ':',
			num: 1,
		}
	}

	if command == "FLUSHDB" { // we are eventually making new maps for both of the maps
		storeMu.Lock()                  // we are not deleting but replacing with new maps, eventually go garbage collector will delete the old maps and free up the memory
		store = make(map[string]string) // for basic string commands
		storeMu.Unlock()

		hashMu.Lock()
		hash = make(map[string]map[string]string) // for hash commands
		hashMu.Unlock()
		return Value{
			typ: '+',
			str: "OK",
		} // this only clears the RAM not the actual aof file that PERSISTS data.
	}

	return Value{}
}
