// server.go (v1)
// cbpcm - 12540054
// cbpcm@mail.missouri.edu
//
// server.go is a TCP socket server for a chatroom
package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

const (
	PORT string = ":10054"
)

var creds []credential // Slice of credentials from users.txt

type credential struct {
	userID string
	password string
}

// initializeUsers opens users.txt and reads each line into a
// new credential struct, then stores it into the creds slice
//
// input: none
// output: none
func initializeUsers() {
	f, err := os.Open("users.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		// Remove leading and trailing parenthesis and split
		// on the delimiter `, `
		cutLeft := strings.TrimLeft(line, "(")
		cutRight := strings.TrimRight(cutLeft, ")")
		userPass := strings.Split(cutRight, ", ")
		// Create new credential
		cred := credential{
			userID: userPass[0],
			password: userPass[1],
		}

		creds = append(creds, cred)
	}
}

// createCredential validates the input(if data is invalid,
// an error will be returned) and creates a new credential
// struct, stores it into the creds slice, and appends it
// to the end of users.txt
//
// input: string, string
// output: error or nil
func createCredential(userID string, password string) error {
	// Data validation
	if len(userID) >= 32 {
		return errors.New("username must be less than 32 characters")
	}
	if len(password) < 4 || len(password) > 8 {
		return errors.New("password must be between 4 and 8 characters")
	}

	// Check if user already exists
	for _, cred := range creds {
		if cred.userID == userID {
			return errors.New("user already exists")
		}
	}

	// Create credential and add to global credentials
	newCred := credential{
		userID: userID,
		password: password,
	}
	creds = append(creds, newCred)

	// Add credential to users.txt
	f, err := os.OpenFile("users.txt", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Printf("Error: cannot open users.txt")
	}
	defer f.Close()
	_, err = f.WriteString("\n("+ userID +", "+ password +")")
	if err != nil {
		log.Printf("Error: cannot write new user to users.txt")
	}

	return nil
}

type client struct {
	connection net.Conn
	name string
	logged bool
	output chan string
	reader *bufio.Reader
	writer *bufio.Writer
}

// new creates and initializes a client pointer. This also
// includes creating the input and output channels and reader
// and writer buffers. The default name is `anon` and the
// default logged value is false
//
// input: net.Conn
// output: none
func new(conn net.Conn) {
	user := &client{
		connection: conn,
		name:       "anon",
		logged:     false,
		output:     make(chan string),
		reader:     bufio.NewReader(conn),
		writer:     bufio.NewWriter(conn),
	}
	user.listen()
}

// listen is a container function to start goroutines
// (concurrency) for the read and write functions of the
// current client
//
// input: none
// output: none
func (client *client) listen() {
	go client.read()
	go client.write()
}

// read takes a client connection's input stream and
// process it into it's respective command and arguments,
// then runs the command action and sends feedback back
// through the clients output channel
//
// input: none
// output: none
func (client *client) read() {
	for {
		message, err := client.reader.ReadString('\n')
		if err != nil {
			log.Printf("Error while reading message: %s", err.Error())
		}
		message = strings.TrimRight(message, "\n")

		args := strings.SplitN(message, " ", 2)
		command := strings.ToLower(args[0])

		switch command {
		case "login":
			args = strings.SplitN(args[1], " ", 2)
			if len(args) != 2 {
				client.output <- "Server: incorrect arguments for command login"
			} else {
				if client.logged {
					client.output <- "Server: you are already logged in"
				} else {
					userID := args[0]
					password := args[1]
					success := client.login(userID, password)
					if success {
						client.output <- "Server: welcome " + client.name + ", your login request was successful"
					} else {
						client.output <- "Server: authentication failed"
					}
				}
			}
		case "newuser":
			args = strings.SplitN(args[1], " ", 2)
			if len(args) != 2 {
				client.output <- "Server: incorrect arguments for command newuser"
			} else {
				if client.logged {
					client.output <- "Server: You are already logged in"
				} else {
					userID := args[0]
					password := args[1]
					err = createCredential(userID, password)
					if err != nil {
						client.output <- "Server: " + err.Error()
					} else {
						success := client.login(userID, password)
						if success {
							client.output <- "Server: welcome " + client.name + ", your login request was successful"
						} else {
							client.output <- "Server: authentication failed"
						}
					}
				}
			}
		case "send":
			if client.logged {
				if len(args) == 2 {
					message := args[1]
					client.output <- client.name + ": " + message
				} else {
					client.output <- "Server: incorrect arguments for command send"
				}
			} else {
				client.output <- "Server: you must be logged in to use this command"
			}
		case "logout":
			if client.logged {
				log.Printf("%s has left. Closing connection.", client.name)
				client.connection.Close()
				return
			} else {
				client.output <- "Server: you must be logged in to use this command"
			}
		default:
			client.output <- "Server: command not found"
		}
	}
}

// write reads the clients output channel and writes
// the data stream to the client
//
// input: none
// output: none
func (client *client) write() {
	for message := range client.output {
		client.writer.WriteString(message+"\n")
		client.writer.Flush()
	}
}

// login compares the parameters to the creds slice
// and sees if there is a match.
// if both parameters match -> return true,
// else -> return false
//
// input: string, string
// output: bool
func (client *client) login(userID string, password string) bool {
	userCreds := credential{
		userID:   userID,
		password: password,
	}
	for _, cred := range creds {
		if userCreds == cred {
			client.name = userID
			client.logged = true
			return true
		}
	}
	return false
}

func main() {
	initializeUsers() // get users from users.txt

	// tcp protocol, listen on PORT
	listener, err := net.Listen("tcp", PORT)
	if err != nil {
	log.Fatalf("Error listening on tcp: %s", err.Error())
	}

	for {
		// accept incoming connections
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %s\n", err.Error())
		}

		// initialize client with new connection
		new(conn)
		fmt.Println("Connection successfully received")
	}
}