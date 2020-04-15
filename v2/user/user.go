// user.go (v2)
// cbpcm - 12540054
// cbpcm@mail.missouri.edu
//
// user.go is a TCP socket client for a chatroom
package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

const PORT string = ":10054"
func main() {
	// tcp dial to server on PORT
	conn, err := net.Dial("tcp", "localhost"+PORT)
	if err != nil {
		log.Fatalf("Error listening on tcp: %s", err.Error())
	}

	defer conn.Close()

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
				fmt.Println("Goodbye!")
				os.Exit(1)
			}
			log.Printf("Error getting message from server: %s", err.Error())
		}

		fmt.Print(message)
	}
}

// output reads the Stdin data stream and writes the
// message line to the connection data stream
func output(conn net.Conn) {
	reader := bufio.NewReader(os.Stdin)
	for {
		text, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error reading message: %s", err.Error())
		}
		fmt.Fprintf(conn, text)
	}
}
