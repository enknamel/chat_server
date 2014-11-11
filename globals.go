package main

import (
	"regexp"
)

//string constants

const nameReq string = "^[a-zA-Z0-9_]{1,10}$"

const welcomeMessage string = `Welcome to the Weeby chat server
Messages are limited to 1kb
You will be disconnected after 10 minutes idle
Type /help for a list of commands after logging in
Login?`

const disconnectMessage string = "Bye"
const floodMessage string = "You are sending too many messages and will be disconnected\n" + disconnectMessage

const badNameMsg string = "Names must match the regex " + nameReq + "\nLogin?"
const takenNameMsg string = "Sorry, name taken.\nLogin?"
const badRoomName string = "Room names must match the regex " + nameReq + "\nLogin?"

//regexes
var validMsgRegex *regexp.Regexp = regexp.MustCompile("[a-zA-Z0-9_()/]{1,}")

var nameRegex *regexp.Regexp = regexp.MustCompile(nameReq)

//package objects

//user registry
var users *userList = newUsersList()

//room registry
var rooms *roomList = newRoomList()
