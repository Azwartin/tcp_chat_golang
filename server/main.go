package main

import (
	"fmt"
	"net"
	"os"
	"tcpchat/server/models"
	"time"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage", os.Args[0], "<host>", "<port>")
		os.Exit(1)
	}

	host := os.Args[1]
	port := os.Args[2]
	service := host + ":" + port
	tcpAddr, err := net.ResolveTCPAddr("tcp", service)
	exitIfError(err)
	listener, err := net.ListenTCP("tcp", tcpAddr)
	exitIfError(err)
	room := models.Room{
		MessageChannel:           make(chan []byte, 20),
		UsersLimit:               50,
		ConnPerUserLimit:         3,
		UserTTL:                  time.Minute * 10,
		UserActivityMonitorDelay: time.Second * 15,
		Users: make(map[string]*models.User),
	}

	fmt.Println("Server started at", service)
	room.Listen(listener)
}

func exitIfError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
