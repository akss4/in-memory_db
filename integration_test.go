package main

import (
	"io"
	"net"
	"sync"
	"testing"
)

func TestTCPServer(t *testing.T) {
	listener, aof, err := startServer("127.0.0.1:0", t.TempDir()+"/test.aof") // Start the server on a random available port
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer listener.Close()
	defer aof.Close()
	go acceptConnections(listener, aof) // go routine to accept connections concurrently

	conn, err := net.Dial("tcp", listener.Addr().String()) // random port to test os assigned
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	command := "*3\r\n$3\r\nSET\r\n$4\r\nname\r\n$5\r\nakash\r\n" // manual resp bcz we are testing tcp not encode function

	_, err = conn.Write([]byte(command))
	if err != nil {
		t.Fatalf("Failed to send command: %v", err)
	}

	response := make([]byte, 1024) // responce of our set  has to be ok

	n, err := conn.Read(response)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if string(response[:n]) != "+OK\r\n" {
		t.Fatalf("Expected +OK, got %q", string(response[:n]))
	}

	getCommand := "*2\r\n$3\r\nGET\r\n$4\r\nname\r\n" // testing get for set command

	_, err = conn.Write([]byte(getCommand))
	if err != nil {
		t.Fatalf("Failed to send GET command: %v", err)
	}

	response = make([]byte, 1024) // has to be akash

	n, err = conn.Read(response)
	if err != nil {
		t.Fatalf("Failed to read GET response: %v", err)
	}

	if string(response[:n]) != "$5\r\nakash\r\n" {
		t.Fatalf("Expected $5\\r\\nakash\\r\\n, got %q", string(response[:n]))
	}

	hsetCommand := "*4\r\n$4\r\nHSET\r\n$4\r\nuser\r\n$4\r\nname\r\n$5\r\nakash\r\n" //testing hash

	_, err = conn.Write([]byte(hsetCommand))
	if err != nil {
		t.Fatalf("Failed to send HSET command: %v", err)
	}

	response = make([]byte, 1024) // has to be ok

	n, err = conn.Read(response)
	if err != nil {
		t.Fatalf("Failed to read HSET response: %v", err)
	}

	if string(response[:n]) != "+OK\r\n" {
		t.Fatalf("Expected +OK, got %q", string(response[:n]))
	}

	hgetCommand := "*3\r\n$4\r\nHGET\r\n$4\r\nuser\r\n$4\r\nname\r\n" // hget test after hset

	_, err = conn.Write([]byte(hgetCommand))
	if err != nil {
		t.Fatalf("Failed to send HGET command: %v", err)
	}

	response = make([]byte, 1024) // shoulf be akash

	n, err = conn.Read(response)
	if err != nil {
		t.Fatalf("Failed to read HGET response: %v", err)
	}

	if string(response[:n]) != "$5\r\nakash\r\n" {
		t.Fatalf("Expected $5\\r\\nakash\\r\\n, got %q", string(response[:n]))
	}
}

func TestTCPMultipleCommands(t *testing.T) {
	listener, aof, err := startServer(
		"127.0.0.1:0",
		t.TempDir()+"/test.aof",
	)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	defer listener.Close()
	defer aof.Close()

	go acceptConnections(listener, aof)

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	commands := "*3\r\n$3\r\nSET\r\n$4\r\nname\r\n$5\r\nakash\r\n" + // set and get
		"*2\r\n$3\r\nGET\r\n$4\r\nname\r\n"

	_, err = conn.Write([]byte(commands))
	if err != nil {
		t.Fatalf("Failed to send commands: %v", err)
	}
	response := make([]byte, 5)

	_, err = io.ReadFull(conn, response)
	if err != nil {
		t.Fatalf("Failed to read SET response: %v", err)
	}

	if string(response) != "+OK\r\n" {
		t.Fatalf("Expected +OK, got %q", string(response))
	}

	response = make([]byte, 11)

	_, err = io.ReadFull(conn, response)
	if err != nil {
		t.Fatalf("Failed to read GET response: %v", err)
	}

	if string(response) != "$5\r\nakash\r\n" {
		t.Fatalf("Expected $5\\r\\nakash\\r\\n, got %q", string(response))
	}
}

func TestTCPFragmentedCommand(t *testing.T) {
	listener, aof, err := startServer(
		"127.0.0.1:0",
		t.TempDir()+"/test.aof",
	)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	defer listener.Close()
	defer aof.Close()

	go acceptConnections(listener, aof)

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	first := "*3\r\n$3\r\nSET\r\n" // half of command

	_, err = conn.Write([]byte(first))
	if err != nil {
		t.Fatalf("Failed to send first fragment: %v", err)
	}

	second := "$4\r\nname\r\n$5\r\nakash\r\n" //half of it

	_, err = conn.Write([]byte(second))
	if err != nil {
		t.Fatalf("Failed to send second fragment: %v", err)
	}

	response := make([]byte, 5)

	_, err = io.ReadFull(conn, response)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if string(response) != "+OK\r\n" {
		t.Fatalf("Expected +OK, got %q", string(response))
	}
}

func TestMultipleClients(t *testing.T) {
	listener, aof, err := startServer(
		"127.0.0.1:0",
		t.TempDir()+"/test.aof",
	)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	defer listener.Close()
	defer aof.Close()

	go acceptConnections(listener, aof)
	conn1, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect client 1: %v", err)
	}
	defer conn1.Close()

	conn2, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect client 2: %v", err)
	}
	defer conn2.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		_, err := conn1.Write([]byte(
			"*3\r\n$3\r\nSET\r\n$4\r\nkey1\r\n$7\r\nclient1\r\n",
		))
		if err != nil {
			t.Error(err)
			return
		}

		response := make([]byte, 5)
		_, err = io.ReadFull(conn1, response)
		if err != nil {
			t.Error(err)
			return
		}

		if string(response) != "+OK\r\n" {
			t.Errorf("expected +OK, got %q", response)
		}
	}()

	go func() {
		defer wg.Done()

		_, err := conn2.Write([]byte(
			"*3\r\n$3\r\nSET\r\n$4\r\nkey2\r\n$7\r\nclient2\r\n",
		))
		if err != nil {
			t.Error(err)
			return
		}

		response := make([]byte, 5)
		_, err = io.ReadFull(conn2, response)
		if err != nil {
			t.Error(err)
			return
		}

		if string(response) != "+OK\r\n" {
			t.Errorf("expected +OK, got %q", response)
		}
	}()

	wg.Wait()
}
