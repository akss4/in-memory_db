package main

import (
	"fmt"
	"net"
	"strings"
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
			fmt.Println("Parsed Value:", value)
			fmt.Println("Bytes consumed:", consumed)

			command := strings.ToUpper(value.array[0].str) // Get the command name from the parsed value
			if command == "SET" || command == "HSET" || command == "HDEL" {
				err := aof.Write(value)
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
			response := handleCommand(value)
			encodedResponse := encode(response)

			_, err = conn.Write(encodedResponse)

			if err != nil {
				fmt.Println("Error writing data:", err)
				break
			}
		}
	}

}

func main() {
	listener, err := net.Listen("tcp", ":6379")
	fmt.Println("Server is listening on port 6379...")
	if err != nil {
		panic(err)
	}
	aof, err := NewAof() // create a new AOF instance after listening on the port, so that we can log the commands to the AOF file
	if err != nil {
		fmt.Println("Error creating AOF:", err)
		panic(err)
	}
	err = aof.Read(func(value Value) { // read the AOF file and replay the commands to restore the state of the database
		handleCommand(value)
	})
	if err != nil {
		fmt.Println("Error reading AOF:", err)
		panic(err)
	}

	defer aof.Close() // close the AOF file when the server is shutting down
	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		go handleConnection(conn, aof) // Handle each connection concurrently
	}
}
