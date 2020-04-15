// client.go (v1)
// cbpcm - 12540054
// cbpcm@mail.missouri.edu
//
// client.go is a TCP socket client for a chatroom
package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

const (
	PORT string = ":10054"
)

func main() {
	// tcp dial to server on PORT
	conn, err := net.Dial("tcp", "localhost"+PORT)
	if err != nil {
		log.Fatalf("Error dailing tcp: %s", err.Error())
		os.Exit(1)
	}

	// start goroutines(concurrency) input and output
	go input(conn)
	for {
		output(conn)
	}
}

// input reads the connection data stream, checks for EOF
// if EOF -> close connection and exit
// else -> print message from server
func input(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("Goodbye...")
				conn.Close()
				os.Exit(0)
			}
			fmt.Printf("Error reading message from server: %s\n", err.Error())
		}

		fmt.Printf("%s", message)
	}
}

// output reads the Stdin data stream and writes the
// message line to the connection data stream
func output(conn net.Conn) {
	reader := bufio.NewReader(os.Stdin)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading message from input: %s\n", err.Error())
		}

		fmt.Fprintf(conn, message)
	}
}