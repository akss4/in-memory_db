package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func handleConnection(conn net.Conn, aof *Aof) {
	defer func() {
		fmt.Println("Closing connection...", conn.RemoteAddr())
		conn.Close()
	}()
	buffer := make([]byte, 1024)
	data := make([]byte, 0)
	for { //outer loop for reading from the connection
		n, err := conn.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println("Error reading data:", err)
			break
		}
		data = append(data, buffer[:n]...)

		for len(data) > 0 { //inner loop for parsing the data
			value, consumed, err := parse(data)
			if err == errIncomplete {
				break // Wait for more data to arrive by actually ending the inner loop and continuing to read from the connection.
			}
			if err != nil {
				fmt.Println("Error parsing data:", err)
				break
			}

			data = data[consumed:] // Remove the consumed bytes from the data slice

			if len(value.array) == 0 {
				fmt.Println("Invalid command: empty array")
				continue
			}

			response, expiry := handleCommand(value) // get sresponce and expiry

			command := strings.ToUpper(value.array[0].str) // Get the command name from the parsed value
			if isWritableCommand(value) {                  // actually writes in aof persists data
				aofCommand := commandForAOF(value, expiry) // aftyer reciving from handle command then only we write in aof
				err := aof.Write(aofCommand)
				if err != nil {
					fmt.Println("Error writing to AOF:", err)
					break
				}
			}

			if command == "FLUSHDB" { // deleting the AOF file when FLUSHDB command is called
				err := aof.Clear()
				if err != nil {
					fmt.Println("Error clearing AOF:", err)
					break
				}
			}

			encodedResponse := encode(response)

			_, err = conn.Write(encodedResponse)

			if err != nil {
				fmt.Println("Error writing data:", err)
				break
			}
		}
	}

}
func startServer(addr string, aofPath string) (net.Listener, *Aof, error) {
	listener, err := net.Listen("tcp", addr)
	fmt.Println("Server is listening on port:", addr)
	if err != nil {
		return nil, nil, err
	}
	aof, err := NewAof(aofPath) // create a new AOF instance after listening on the port, so that we can log the commands to the AOF file
	if err != nil {
		fmt.Println("Error creating AOF:", err)
		return nil, nil, err
	}
	err = aof.Read(func(value Value) { // read the AOF file and replay the commands to restore the state of the database
		handleCommand(value)
	})
	if err != nil {
		fmt.Println("Error reading AOF:", err)
		return nil, nil, err
	}

	return listener, aof, nil // newaof is already a pointer
}

func acceptConnections(listener net.Listener, aof *Aof) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}

		go handleConnection(conn, aof)
	}
}

// ttl expiration background goroutine
func startExpirationCleaner() {
	ticker := time.NewTicker(time.Second) // create a ticker that ticks every second
	defer ticker.Stop()                   // stop the ticker when the function returns
	for range ticker.C {                  // for each tick, we check the expiration map for expired keys and delete them from the store and expiration map

		type expiredKey struct {
			key    string
			expiry time.Time
		}
		expiredKeys := make([]expiredKey, 0)
		expirationMu.RLock()
		for key, expiry := range expiration { // we check each value in map and if it is expired, we delete it from the map. We use a read lock to avoid blocking other goroutines that are reading from the map.
			if time.Now().After(expiry) { // checking if expiration is due or not
				expiredKeys = append(expiredKeys, expiredKey{key: key, expiry: expiry}) // store it in
			}
		}
		expirationMu.RUnlock()
		for _, item := range expiredKeys { // we range it in order to delete the expired keys from the store and expiration map. We use a write lock to avoid blocking other goroutines that are writing to the map.
			storeMu.Lock()
			expirationMu.Lock() // tjhis is to avoid race condition when we are, a specific pattern that store is locked 1st and expiration after tha

			currentExpiry, ok := expiration[item.key]
			if ok && currentExpiry.Equal(item.expiry) {

				delete(store, item.key)
				delete(expiration, item.key)
			}

			expirationMu.Unlock()
			storeMu.Unlock()
		}

	}
}

func main() {
	listener, aof, err := startServer(":6379", "./data/database.aof")
	if err != nil {
		fmt.Println("Error starting server:", err)
		panic(err)
	}
	go startExpirationCleaner() // start the expiration cleaner goroutine

	defer listener.Close()
	defer aof.Close()

	go acceptConnections(listener, aof)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan

	fmt.Println("Shutting down server...") // gracaefull shutdown

	fmt.Println("Closing connections...")

	listener.Close()
	aof.Close()
}
