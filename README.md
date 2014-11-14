Chat Server
===========

This is a simple chat server written in Go. It supports multiple users chatting in rooms and private messaging.

Users may join or create rooms as they wish

The executable has two flags, -ip for the listen ip and -port for the listen port.

There is a goroutine for each user and a goroutine for each room.
