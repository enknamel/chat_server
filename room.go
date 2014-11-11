package main

import (
	"bytes"
	"fmt"
	"sort"
	"sync"
)

//room represents a chat room
//a chat room is a named collection of users with a channel mux to all users in the room
type room struct {
	users map[string]*user //user name to user lobby
	msgs  chan *userMsg
	name  string
	sync.Mutex
}

func (r *room) handleMsg(u *user, m string) {
	r.msgs <- &userMsg{u, m, false}
}

//String representation of the room outputs all the users in the room
func (r *room) String(u *user) string {
	buffer := bytes.Buffer{}

	names := make([]string, 0)

	for name, _ := range r.users {
		names = append(names, name)
	}

	sort.Strings(names)

	for _, name := range names {
		buffer.WriteString("* " + name)
		if name == u.name {
			buffer.WriteString(" (** this is you)")
		}
		buffer.WriteString("\n")
	}

	buffer.WriteString("end of list")

	return buffer.String()
}

//adding a user to the room announces to the user he is joining the room
//announces to the room (excluding the joiner) that the user joined
func (r *room) addUser(u *user) {
	r.Lock()
	defer r.Unlock()

	r.users[u.name] = u
	u.msgs <- fmt.Sprintf("entering room: %s", r.name)
	u.msgs <- r.String(u)
	r.msgs <- &userMsg{u, "* new user joined chat: " + u.name, true}
}

//removes a user from the room
//mutex is held by the caller
func (r *room) removeUser(u *user) {
	delete(r.users, u.name)
	u.room = nil
}

//goroutine for handling passing messages to users in a room
func (r *room) run() {

	defer r.Unlock()

	for msg := range r.msgs {
		//send message to all users except sender
		r.Lock()
		for _, u := range r.users {
			if u != msg.u {
				if msg.a {
					u.msgs <- msg.m
				} else {
					u.msgs <- u.name + ": " + msg.m
				}
			}
		}
		r.Unlock()
	}
}

//should only be called by roomList to handle removing a room entirely when it has no users left
//mutex is already held by the caller so no need to worry about it
func (r *room) destory() {
	close(r.msgs)
}

//constructor for room
func newRoom(name string) *room {
	r := new(room)
	r.users = make(map[string]*user)
	r.msgs = make(chan *userMsg)
	r.name = name
	go r.run()
	return r
}

//registry of rooms
type roomList struct {
	rooms map[string]*room
	count int
	sync.Mutex
}

//used by /join to either join an existing room or create it if it does not exist
func (r *roomList) userJoinOrCreateRoom(u *user, roomName string) {
	r.Lock()
	defer r.Unlock()
	if !nameRegex.MatchString(roomName) {
		u.msgs <- badRoomName
		return
	}
	if _, ok := r.rooms[roomName]; !ok {
		room := newRoom(roomName)
		r.rooms[roomName] = room
	}

	r.rooms[roomName].addUser(u)
	u.room = r.rooms[roomName]
}

//removes a user from the room. if the room now has zero users the room is closed
func (r *roomList) removeUser(u *user) {
	r.Lock()
	defer r.Unlock()

	room := u.room

	if room == nil {
		return
	}

	//this mutex needs to be held across multiple operations on the room
	//so it is locked here instead of locked within the room
	room.Lock()
	defer room.Unlock()

	room.removeUser(u)

	if len(room.users) == 0 {
		delete(r.rooms, room.name)
		room.destory()
	} else {
		room.msgs <- &userMsg{u, "user has left chat: " + u.name, true}
	}

}

//string representation of a roomList displays as the list of all rooms with how many users are in them
func (r *roomList) String() string {
	r.Lock()
	defer r.Unlock()
	if len(r.rooms) == 0 {
		return "There are currently no rooms. Use /join to create one!"
	}
	msgs := bytes.Buffer{}
	roomNames := make([]string, 0)
	msgs.WriteString("Active rooms are:\n")
	for roomName, _ := range r.rooms {
		roomNames = append(roomNames, roomName)
	}
	sort.Strings(roomNames)

	for _, roomName := range roomNames {
		room := r.rooms[roomName]
		msgs.WriteString(fmt.Sprintf("%s (%d)\n", room.name, len(room.users)))
	}
	msgs.WriteString("end of list")
	return msgs.String()
}

//constructor for roomList
func newRoomList() *roomList {
	r := new(roomList)
	r.rooms = make(map[string]*room)
	return r
}
