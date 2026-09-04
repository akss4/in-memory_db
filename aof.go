package main

import (
	"bufio"
	"io"
	"os"
	"sync"
	"time"
)

type Aof struct {
	file      *os.File
	rd        *bufio.Reader
	mu        sync.Mutex
	done      chan struct{}
	wg        sync.WaitGroup
	closeOnce sync.Once
}

func NewAof(path string) (*Aof, error) {
	file, err := os.OpenFile(path, //NAME OF FILE
		os.O_CREATE|os.O_RDWR|os.O_APPEND, // CREATE FILE IF NOT EXISTS, READ AND WRITE, APPEND MODE
		0644)
	if err != nil {
		return nil, err
	}
	rd := bufio.NewReader(file)
	aof := &Aof{
		file: file,
		rd:   rd,
		done: make(chan struct{}),
	}
	aof.wg.Add(1) //

	go func() {
		defer aof.wg.Done()

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				aof.mu.Lock()
				_ = aof.file.Sync()
				aof.mu.Unlock()

			case <-aof.done:
				return
			}
		}
	}()
	return aof, nil
}

func (aof *Aof) Write(value Value) error {
	aof.mu.Lock()
	defer aof.mu.Unlock()             // no matter what happens, we need to unlock the mutex, so we use defer to unlock the mutex at the end of the function
	encoded := encode(value)          // changing the value to RESP format using encode function
	_, err := aof.file.Write(encoded) // writing the encoded value to the file
	if err != nil {
		return err
	}
	return nil
}
func (aof *Aof) Read(callback func(value Value)) error {
	_, err := aof.file.Seek(0, 0) // Move the file pointer to the beginning of the file
	if err != nil {
		return err
	}
	data, err := io.ReadAll(aof.file) // Read the entire file into memory
	for len(data) > 0 {               // Parse each value from the data
		value, consumed, err := parse(data)
		if err != nil {
			return err
		}
		data = data[consumed:] // Remove the consumed bytes from the data slice
		callback(value)        // Call the callback function with the parsed value
	}
	return nil
}

func (aof *Aof) Clear() error {
	aof.mu.Lock()
	defer aof.mu.Unlock()
	err := aof.file.Truncate(0) // Truncate the file to zero length
	if err != nil {
		return err
	}
	_, err = aof.file.Seek(0, 0) // Move the file pointer to the beginning of the file
	if err != nil {
		return err
	}
	return nil
}
func (aof *Aof) Close() error {
	var err error

	aof.closeOnce.Do(func() {
		close(aof.done)

		aof.wg.Wait()

		aof.mu.Lock()
		defer aof.mu.Unlock()

		if syncErr := aof.file.Sync(); syncErr != nil {
			err = syncErr
			return
		}

		err = aof.file.Close()
	})

	return err
}
