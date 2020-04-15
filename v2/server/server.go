// server.go (v2)
// cbpcm - 12540054
// cbpcm@mail.missouri.edu
//
// server.go is a TCP socket server for a chatroom
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

const (
	// MAXCLIENTS restricts the number of connections that can be
	// active at the same time
	MAXCLIENTS int = 3
	// PORT is the tcp listening port
	PORT string = ":10054"
)
var list []Credentials

type Credentials struct {
	Username string
	Password string
}

// loadUsers opens users.txt and reads each line into a
// new credential struct, then stores it into the creds slice
//
// input: none
// output: none
func loadUsers() {
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
		creds := strings.Split(cutRight, ", ")
		// Create new credential
		user := Credentials{
			Username: creds[0],
			Password: creds[1],
		}
		list = append(list, user)
	}
}

// addUserCredentials validates the input(if data is invalid,
// an error will be returned) and creates a new credential
// struct, stores it into the creds slice, and appends it
// to the end of users.txt
//
// input: string, string
// output: error or nil
func addUserCredentials(username string, password string) error {
	if len(username) >= 32 {
		return errors.New("Username must be less than 32 characters.")
	}
	if len(password) < 4 || len(password) > 8 {
		return errors.New("Password must be between 4 and 8 characters.")
	}
	for _, cred := range list {
		if cred.Username == username {
			return errors.New("User already exists.")
		}
	}
	newuser := Credentials{
		Username: username,
		Password: password,
	}
	list = append(list, newuser)

	f, err := os.OpenFile("users.txt", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Printf("Error: cannot open users.txt")
	}
	defer f.Close()
	_, err = f.WriteString("\n("+ username +", "+ password +")")
	if err != nil {
		log.Printf("Error: cannot write new user to users.txt")
	}
	return nil
}

// checkCredentials compares the parameters to the creds
// slice and sees if there is a match
// if both parameters match -> return true,
// else -> return false
//
// input: string, string
// output: bool
func checkCredentials(username string, password string) bool {
	for _, cred := range list {
		if cred.Username == username {
			if cred.Password == password {
				return true
			}
		}
	}
	return false
}

type Client struct {
	name   string
	logged bool
	conn net.Conn
	in     chan string
	out    chan string
	reader *bufio.Reader
	writer *bufio.Writer
}

// Read takes a client connection's input stream and
// process it into it's respective command and arguments,
// then runs the command action and sends feedback back
// through the clients output channel
//
// input: none
// output: none
func (client *Client) Read() {
	for {
		message, _ := client.reader.ReadString('\n')
		err := processCommand(client, message)
		log.Print("Recieved command: " + message)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("Error while processing command: %s", err.Error())
		}

	}
}

// Write reads the clients output channel and writes
// the data stream to the client
//
// input: none
// output: none
func (client *Client) Write() {
	for data := range client.out {
		client.writer.WriteString(data)
		client.writer.Flush()
	}
}

// Listen is a container function to start goroutines
// (concurrency) for the read and write functions of the
// current client
//
// input: none
// output: none
func (client *Client) Listen() {
	go client.Read()
	go client.Write()
}

// New creates and initializes a client pointer. This also
// includes creating the input and output channels and reader
// and writer buffers. The default name is `anon` and the
// default logged value is false
//
// input: net.Conn
// output: none
func New(conn net.Conn) *Client {
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	client := &Client{
		name:   "anon",
		logged: false,
		conn: conn,
		in:     make(chan string),
		out:    make(chan string),
		reader: reader,
		writer: writer,
	}
	client.Listen()

	return client
}

// processCommand takes a client and their message and
// processes it into it's respective command and arguments,
// then runs the command action and sends feedback back
//
// input: *Client, string
// output: error or nil
func processCommand(client *Client, message string) error {
	splitMessage := strings.SplitN(strings.TrimRight(message, "\n"), " ", 3)

	var command string
	var argOne string
	var argTwo string

	command = splitMessage[0]

	// Data validation of message
	if len(splitMessage) > 3 {
		return errors.New("invalid command")
	} else if len(splitMessage) == 2 {
		return errors.New("invalid command")
	}
	if len(splitMessage) > 1 {
		argOne = splitMessage[1]
		argTwo = splitMessage[2]
	}

	// Select correct action from command input
	switch command {
	case "who":
		if client.logged {
			client.in <- "who " + client.name // send who command to chat board
		} else {
			client.out <- "Server: Please log in first.\n"
			return errors.New("client not logged in")
		}
	case "login":
		if checkCredentials(argOne, argTwo) {
			if client.logged == false {
				client.logged = true
				client.name = argOne
				log.Println(argOne + " logged in successfully.")
				// send welcome message to client
				client.out <- "Server: Welcome " + argOne + ", your login request was successful.\n"
				// send loggedin action to chat board
				client.in <- "loggedin " + argOne
			} else {
				client.out <- "Server: You are already logged in.\n"
			}
		} else {
			client.out <- "Server: Authentication failed. Please try again.\n"
			return errors.New("authentication failed")
		}
	case "newuser":
		addUserCredentials(argOne, argTwo)
		client.logged = true
		client.name = argOne
		// send welcome message to client
		client.out <- "Server: Welcome " + argOne + ", you are currently logged in.\n"
		// send loggedin action to chat board
		client.in <- "loggedin " + argOne
	case "send":
		if client.logged {
			// send command and message to chat board
			client.in <- command +" "+ argOne +" "+ client.name +": "+ argTwo
		} else {
			client.out <- "Server: please log in first.\n"
			return errors.New("client not logged in")
		}
	case "logout":
		if client.logged {
			// send logout command to chat board, close connection, return EOF
			client.in <- "logout " + client.name
			client.conn.Close()
			return io.EOF
		}
		client.out <- "Server: Please log in first.\n"
		return errors.New("client not logged in")
	default:
		// if the command isn't valid, send them feedback
		client.out <- "Server: Command not found.\n"
		return errors.New("command not found")
	}
	// if there are no errors, return nil
	return nil
}

type ChatBoard struct {
	clients     []*Client
	connections chan net.Conn
	in          chan string
	out         chan string
}

// NewChat creates and initialize a chat board with
// input and output streams, connection stream, and
// a client slice
//
// input: none
// output: none
func NewChat() *ChatBoard {
	chatBoard := &ChatBoard{
		clients:     make([]*Client, 0),
		connections: make(chan net.Conn),
		in:          make(chan string),
		out:         make(chan string),
	}
	chatBoard.Listen()

	return chatBoard
}

// Connect initializes a new client, adds it to a
// client slice, and adds the clients output of the
// input stream to the chat boards input stream
//
// input: net.Conn
// output: none
func (chatBoard *ChatBoard) Connect(conn net.Conn) {
	client := New(conn)
	chatBoard.clients = append(chatBoard.clients, client)
	go func() {
		for {
			chatBoard.in <- <-client.in
		}
	}()
}

// SendMessage reads the chat board's input stream
// and processes the command and arguments passed in
// it. This is weather all messages are sent out to
// clients
//
// input: string
// output: none
func (chatBoard *ChatBoard) SendMessage(message string) {
	msg := strings.SplitN(strings.TrimRight(message, "\n"), " ", 4)
	
	if msg[0] == "who" {
		var currentUser *Client
		var names []string

		for _, client := range chatBoard.clients {
			if client.name == msg[1] {
				currentUser = client
			} 
			if client.logged {
				names = append(names, client.name)
			}
		}
		currentUser.out <- "Current online users\n"
		currentUser.out <- "--------------------\n"
		for _, name := range names {
			fmt.Println(name)
			currentUser.out <- name+"\n"
		}
	} else if msg[0] == "send" {
		if msg[1] == "all" { // Braodcast
			for _, client := range chatBoard.clients {
				client.out <- msg[2] +" "+ msg[3] +"\n"
			}
		} else { // Unicast
			for _, client := range chatBoard.clients {
				if client.name == msg[1] {
					client.out <- msg[2] +" "+ msg[3] +"\n"
				}
			}
		}
	} else if msg[0] == "logout" {
		for i, client := range chatBoard.clients {
			if client.name == msg[1] {
				chatBoard.clients = append(chatBoard.clients[:i], chatBoard.clients[i+1:]...)
			}
			client.out <- msg[1] + " has left the chat.\n"
		}
	} else if msg[0] == "loggedin" {
		for _, client := range chatBoard.clients {
			if client.logged {
				client.out <- msg[1] + " has joined the chat.\n"
			}
		}
	}
}

// Listen listens to the chat board's input and
// connections stream and acts depending on which
// is received
// if input stream -> SendMessage
// if connections -> Connect
//
// input: none
// output: none
func (chatBoard *ChatBoard) Listen() {
	go func() {
		for {
			select {
			case data := <-chatBoard.in:
				chatBoard.SendMessage(data)
			case conn := <-chatBoard.connections:
				chatBoard.Connect(conn)
			}
		}
	}()
}

func main() {
	loadUsers() // initialize user's credentials list
	chat := NewChat() // initialize chat board

	// tcp protocol listen on PORT
	listener, err := net.Listen("tcp", PORT)
	if err != nil {
		log.Fatalf("Error listening on tcp: %s", err.Error())
	}

	for {
		// listen for incoming tcp connections if current
		// connection is less than MAXCLIENTS
		if len(chat.clients) < MAXCLIENTS {
			conn, err := listener.Accept()
			if err != nil {
				log.Fatalf("Error accepting connection: %s", err.Error())
			}
			fmt.Println("Connection recieved successfully")
			// Add current connection to connection channel
			chat.connections <- conn
		} else {
			log.Println("Connection refused: MAXCLIENTS limit reached")
		}
	}
}
