package main

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

var store = make(map[string]string)
var storeMu = sync.RWMutex{} // for basic string commands

var hash = make(map[string]map[string]string)
var hashMu = sync.RWMutex{} // for hash commands

var expiration = make(map[string]time.Time)
var expirationMu = sync.RWMutex{} // for expiration commands

func handleCommand(value Value) (Value, *time.Time) {
	if value.typ != '*' {
		return Value{}, nil
	}
	if len(value.array) == 0 {
		return Value{}, nil
	}
	if value.array[0].typ != '$' {
		return Value{}, nil
	}
	command := value.array[0].str
	command = strings.ToUpper(command)

	if command == "PING" {
		if len(value.array) > 1 {
			return Value{
				typ: '+',
				str: value.array[1].str,
			}, nil
		}
		return Value{
			typ: '+',
			str: "PONG",
		}, nil
	}
	if command == "SET" {
		var expiry *time.Time
		if len(value.array) != 3 && len(value.array) != 5 {
			return Value{
				typ: '-',
				str: "ERR wrong number of argument for 'SET' command",
			}, nil
		}
		if len(value.array) == 5 {
			if strings.ToUpper(value.array[3].str) != "EX" {
				return Value{
					typ: '-',
					str: "ERR syntax error",
				}, nil
			}
			seconds, err := strconv.Atoi(value.array[4].str)
			if err != nil {
				return Value{
					typ: '-',
					str: "ERR invalid expire time",
				}, nil
			}

			calculatedExpiry := calculateExpiration(seconds)
			expiry = &calculatedExpiry

			expirationMu.Lock()
			expiration[value.array[1].str] = calculatedExpiry
			expirationMu.Unlock()
		}

		key := value.array[1].str
		val := value.array[2].str
		if len(value.array) == 3 {
			expirationMu.Lock()
			delete(expiration, key) // remove expiration if it exists because 3 value persist command
			expirationMu.Unlock()
		}
		storeMu.Lock()
		store[key] = val
		storeMu.Unlock()
		return Value{
			typ: '+',
			str: "OK",
		}, expiry
	}

	if command == "GET" {
		if len(value.array) != 2 {
			return Value{
				typ: '-',
				str: "ERR wrong number of argument for 'GET' command",
			}, nil

		}
		key := value.array[1].str

		expirationMu.RLock()                             // checking expiration
		expirationTime, hasExpiration := expiration[key] // has expirration is coma ok syntax its a bool and expiration time is normal value
		expirationMu.RUnlock()
		if hasExpiration && time.Now().After(expirationTime) {
			storeMu.Lock()
			expirationMu.Lock()
			delete(store, key)
			delete(expiration, key)
			storeMu.Unlock()
			expirationMu.Unlock()
			return Value{
				typ: '$',
				str: "",
			}, nil
		}
		storeMu.RLock()
		val, ok := store[key]
		storeMu.RUnlock()
		if !ok {
			return Value{
				typ: '$',
				str: "",
			}, nil
		}
		return Value{
			typ: '$',
			str: val, // get returns the value of the key if it exists, otherwise it returns an empty string
		}, nil
	}

	if command == "HSET" {
		if len(value.array) != 4 {
			return Value{
				typ: '-',
				str: "ERR wrong number of argument for 'HSET' command",
			}, nil
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
		}, nil

	}

	if command == "HGET" {
		if len(value.array) != 3 {
			return Value{
				typ: '-',
				str: "ERR wrong number of argument for 'HGET' command",
			}, nil
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
			}, nil
		}
		return Value{
			typ: '$',
			str: val,
		}, nil
	}

	if command == "HGETALL" {
		if len(value.array) != 2 {
			return Value{
				typ: '-',
				str: "ERR wrong number of argument for 'HGETALL' command",
			}, nil
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
		}, nil
	}

	if command == "HDEL" {
		if len(value.array) != 3 {
			return Value{
				typ: '-',
				str: "ERR wrong number of argument for 'HDEL' command",
			}, nil
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
			}, nil
		}
		delete(hash[key], field)
		hashMu.Unlock()
		return Value{
			typ: ':',
			num: 1,
		}, nil
	}

	if command == "FLUSHDB" { // we are eventually making new maps for both of the maps
		storeMu.Lock()                  // we are not deleting but replacing with new maps, eventually go garbage collector will delete the old maps and free up the memory
		store = make(map[string]string) // for basic string commands
		storeMu.Unlock()

		hashMu.Lock()
		hash = make(map[string]map[string]string) // for hash commands
		hashMu.Unlock()

		expirationMu.Lock()
		expiration = make(map[string]time.Time) // for expiration commands
		expirationMu.Unlock()
		return Value{
			typ: '+',
			str: "OK",
		}, nil // this only clears the RAM not the actual aof file that PERSISTS data.
	}

	return Value{}, nil
}

func isWritableCommand(value Value) bool { /// aof wrotye without check if its valid or not soo we fixed tht
	if value.typ != '*' || len(value.array) == 0 || value.array[0].typ != '$' {
		return false
	}

	command := strings.ToUpper(value.array[0].str)

	switch command {
	case "SET":
		return len(value.array) == 3 || len(value.array) == 5
	case "HSET":
		return len(value.array) == 4
	case "HDEL":
		return len(value.array) == 3
	default:
		return false
	}
}

// a unified function to check expirartion that will be further used for aof too

func calculateExpiration(seconds int) time.Time {
	return time.Now().Add(time.Duration(seconds) * time.Second)
}

func commandForAOF(value Value, expiry *time.Time) Value {
	if expiry == nil {
		return value
	}
	timestamp := strconv.FormatInt(expiry.Unix(), 10)
	aofCommand := Value{
		typ: '*', // custom value with ex xchanged to exat rather than the original one
		array: []Value{
			{typ: '$', str: "SET"},
			value.array[1], // raw taking 1 and 2 from the original value
			value.array[2],
			{typ: '$', str: "EXAT"},    // ex changed to exat will be used for aof persistance and absolute time stamp when it wil be expired
			{typ: '$', str: timestamp}, // timestamp which is chasnged to unix expiry time by strconv
		},
	}
	return aofCommand
}
