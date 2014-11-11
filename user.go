package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"
)

//represents a user. users have a name, belong to exactly 1 room and receive messages
type user struct {
	name                 string
	room                 *room
	msgs                 chan string
	lastMessagedUserName string //support for /rm
	lastReceivedUserName string //support for /r
	deleted              bool
	sync.Mutex
}

//dto for messages to be received by users. contains sender/actioned user, message, and a channel announce flag
type userMsg struct {
	u *user
	m string
	a bool
}

func (u *user) loggedIn() bool {
	return u.name != ""
}

//setting a name on a user changes them to the logged in state
//logging in successfully requires choosing a valid not taken name
func (u *user) setName(n string) {
	if nameRegex.MatchString(n) {
		if users.addUser(u, n) {
			u.name = n
			u.msgs <- "Welcome " + n + "!"
		} else {
			u.msgs <- takenNameMsg
		}
	} else {
		u.msgs <- badNameMsg
	}
}

//process messages sent by a user
func (u *user) sendMessage(msg string) {
	msgs := strings.Split(msg, "\r\n")
	for _, m := range msgs {
		if len(m) > 0 {
			if m[0] == '/' {
				u.processCommand(m)
			} else if u.room != nil {
				u.chat(m)
			} else {
				u.msgs <- "* You need to join a room to chat. You can still private message users"
			}
		}
	}
}

func (u *user) receivePrivateMessage(sender *user, msg string) {
	u.Lock()
	defer u.Unlock()

	u.lastReceivedUserName = sender.name

	u.msgs <- fmt.Sprintf("(*pm %s) %s", sender.name, msg)
}

func (u *user) leaveRoom() {
	u.Lock()
	defer u.Unlock()
	rooms.removeUser(u)
}

//process all / commands
func (u *user) processCommand(msg string) {
	for {
		switch {
		case strings.Index(msg, "/join") == 0:
			parts := strings.Split(msg, " ")
			if len(parts) != 2 {
				break
			}
			rooms.userJoinOrCreateRoom(u, parts[1])
			return
		case msg == "/rooms":
			fmt.Println(rooms.String())
			u.msgs <- rooms.String()
			return
		case msg == "/leave":
			if u.room == nil {
				break
			}
			u.leaveRoom()
			return
		case msg == "/quit":
			u.destroy()
			return
		case strings.Index(msg, "/m ") == 0:
			firstSpace := strings.Index(msg[3:], " ")
			if firstSpace == -1 || 3+firstSpace >= len(msg)-1 {
				break
			}

			recipient := msg[3 : 3+firstSpace]

			users.privateMessage(u, recipient, msg[3+firstSpace+1:])
			return
		case strings.Index(msg, "/r ") == 0 && len(msg) > 3:
			if u.lastReceivedUserName == "" {
				u.msgs <- "* You need to receive a message to send a reply"
				return
			}
			msg = msg[3:]
			users.privateMessage(u, u.lastReceivedUserName, msg)
			return
		case strings.Index(msg, "/rm ") == 0 && len(msg) > 4:
			if u.lastMessagedUserName == "" {
				u.msgs <- "* You need to send a message to re-message a user"
				return
			}
			msg = msg[4:]
			users.privateMessage(u, u.lastMessagedUserName, msg)
			return
		case msg == "/users":
			u.msgs <- users.String(u)
			return
		case msg == "/help":
			break
		default:
			u.msgs <- "* Unrecognized command"
			break
		}

		break
	}
	//failure print help
	u.msgs <- printHelp()
}

//returns the help text
func printHelp() string {
	return `* Available commands:
* /help Display this help message.
* /leave Leave the room you are currently in. You will need to join another room to continue chatting.
* /m [user] [message] send a private message to the specified user
* /join [room] you will join the specified room
* /quit leave the server and disconnect
* /r [message] reply to the user who last messaged you
* /rooms list all available chat rooms
* /rm [message] send a message to the user you last message
* /users list all logged in users`
}

//send a message to the user's chat room
func (u *user) chat(msg string) {
	if u.room != nil {
		u.room.handleMsg(u, msg)
	}
}

//clean up user and remove them from the system
func (u *user) destroy() {
	u.Lock()
	defer u.Unlock()
	if !u.deleted {
		rooms.removeUser(u)
		users.removeUser(u)
		u.msgs <- "BYE"
		close(u.msgs)
		u.deleted = true
	}
}

//constructor for user
func newUser() *user {
	u := new(user)
	u.msgs = make(chan string, 128)
	return u
}

//global registry of all users. used to control access of user message channels
type userList struct {
	users map[string]*user
	sync.Mutex
}

//handle adding a user to the registry. names must be unique
//returns success if a user is added to the registry
func (ul *userList) addUser(u *user, proposedName string) bool {
	ul.Lock()
	defer ul.Unlock()

	if _, ok := ul.users[proposedName]; !ok {
		ul.users[proposedName] = u
		return true
	}

	return false
}

func (ul *userList) privateMessage(u *user, recipientName string, msg string) {
	ul.Lock()
	defer ul.Unlock()
	u.Lock()
	defer u.Unlock()

	if u.name == recipientName {
		u.msgs <- "* talking to yourself?"
	} else if recipient, ok := ul.users[recipientName]; ok {
		recipient.receivePrivateMessage(u, msg)
		u.lastMessagedUserName = recipientName
	} else {
		u.msgs <- "* Could not find user: " + recipientName
	}
}

//removes a user from the registry. should only be called from user.destroy
func (ul *userList) removeUser(u *user) {
	ul.Lock()
	defer ul.Unlock()

	delete(ul.users, u.name)
}

func (ul *userList) String(u *user) string {
	ul.Lock()
	defer ul.Unlock()

	names := make([]string, 0)
	for name, _ := range ul.users {
		names = append(names, name)
	}

	sort.Strings(names)
	result := bytes.Buffer{}
	result.WriteString("Logged in users\n")
	for _, name := range names {
		result.WriteString("* ")
		result.WriteString(name)
		if name == u.name {
			result.WriteString(" (** this is you)")
		}
		result.WriteString("\n")
	}
	result.WriteString("end of list")

	return result.String()
}

//constructor for userList
func newUsersList() *userList {
	u := new(userList)
	u.users = make(map[string]*user)
	return u
}

//goroutine for processing user connections
//when a connection is closed by either side the user is destroyed
func processUser(con *net.TCPConn) {

	defer con.Close()

	fmt.Fprintln(con, welcomeMessage)

	u := newUser()

	defer u.destroy()

	//2k byte buffer. Messages limited to 1k
	buf := make([]byte, 2048)

	liveChan := make(chan struct{}, 1)

	//10 minute activity timeout
	timeout := time.NewTimer(10 * time.Minute)
	go func() {
		defer func() {
			if e := recover(); e != nil {
				fmt.Println(e, string(debug.Stack()))
			}
		}()

		for {
			liveChan <- struct{}{}
			b, err := con.Read(buf)
			if err == io.EOF {
				return
			} else {
				p(err)
			}
			fmt.Println("New Message!")
			msg := string(buf[:b])
			fmt.Println(b, msg)

			//user doing something weird
			//or blank lines
			if b <= 2 {
				continue
			}

			//1k cut off
			if b > 1026 {
				fmt.Fprintln(con, floodMessage)
				return
			}

			//telnet seems to use \r\n so require that as message ending
			if msg[b-2:b] != "\r\n" {
				fmt.Fprintln(con, "Please end all messages in a newline")
				continue
			}

			if !u.loggedIn() {
				u.setName(msg[:len(msg)-2])
			} else {
				u.sendMessage(msg)
			}
		}
	}()

	for {
		select {
		case msg, ok := <-u.msgs:
			if !ok { //user has disconnected
				return
			}
			fmt.Fprintln(con, msg)
		case <-liveChan:
			timeout.Reset(10 * time.Minute)
		case <-timeout.C:
			fmt.Fprintln(con, "Disconnected for being idle")
			return
		}
	}
}
