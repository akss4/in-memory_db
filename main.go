package main

import (
	"fmt"
	"io"
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
			fmt.Println("Parsed Value:", value)
			fmt.Println("Bytes consumed:", consumed)

			if len(value.array) == 0 {
				fmt.Println("Invalid command: empty array")
				continue
			}

			command := strings.ToUpper(value.array[0].str) // Get the command name from the parsed value
			if isWritableCommand(value) {
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

func main() {
	listener, aof, err := startServer(":6379", "/app/data/database.aof")
	if err != nil {
		fmt.Println("Error starting server:", err)
		panic(err)
	}

	defer listener.Close()
	defer aof.Close()

	acceptConnections(listener, aof)
}
