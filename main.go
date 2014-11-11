package main

import (
	"net"
	//"time"
)

func p(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {

	//start tcp server

	listenAddr, err := net.ResolveTCPAddr("tcp", ":9399")

	p(err)

	tcpCon, err := net.ListenTCP("tcp", listenAddr)

	p(err)

	for {
		con, err := tcpCon.AcceptTCP()

		p(err)

		go processUser(con)
	}
}
