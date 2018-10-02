package main

import (
	"fmt"
	"net"
	"os"
	"tcpchat/client/chat"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage", os.Args[0], "host", "port")
		os.Exit(1)
	}

	host := os.Args[1]
	port := os.Args[2]
	service := host + ":" + port
	conn, err := net.Dial("tcp", service)
	exitIfError(err)
	err = chat.Start(conn)
	exitIfError(err)
}

func exitIfError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
