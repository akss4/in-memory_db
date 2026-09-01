package main

import (
	"fmt"
	"net"
)

func handleConnection(conn net.Conn) {
	defer func() {
		fmt.Println("Closing connection...", conn.RemoteAddr())
		conn.Close()
	}()
	buffer := make([]byte, 1024)
	data := make([]byte, 0)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Println("Error reading data:", err)
			break
		}
		data = append(data, buffer[:n]...)
		value, consumed, err := parse(data)
		if err == errIncomplete {
			continue // Wait for more data to arrive
		}
		if err != nil {
			fmt.Println("Error parsing data:", err)
			break
		}
		data = data[consumed:] // Remove the consumed bytes from the data slice
		fmt.Println("Parsed Value:", value)
		fmt.Println("Bytes consumed:", consumed)

		message := string(buffer[:n])
		fmt.Println("Received message:", message)

		_, err = conn.Write([]byte("Message received: " + message))
		if err != nil {
			fmt.Println("Error writing data:", err)
			break
		}
	}

}

func main() {
	listener, err := net.Listen("tcp", ":6379")
	fmt.Println("Server is listening on port 6379...")
	if err != nil {
		panic(err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		go handleConnection(conn)
	}
}
