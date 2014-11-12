package main

import (
	"flag"
	"net"
	//"time"
)

func p(e error) {
	if e != nil {
		panic(e)
	}
}

var listenIp *string = flag.String("ip", "", "specify the ip to listen on")
var listenPort *string = flag.String("port", "9399", "specify the port to listen on")

func main() {

	//start tcp server

	flag.Parse()

	listenAddr, err := net.ResolveTCPAddr("tcp", *listenIp+":"+*listenPort)

	p(err)

	tcpCon, err := net.ListenTCP("tcp", listenAddr)

	p(err)

	for {
		con, err := tcpCon.AcceptTCP()

		p(err)

		go processUser(con)
	}
}
