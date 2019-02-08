package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/faroyam/udp-hole-punch/client"
	"github.com/faroyam/udp-hole-punch/server"

	"go.uber.org/zap"
)

var (
	serverAddr string
	serverPort string
	serverMode bool
	keepalive  bool
	localID    string
	remoteID   string
	timeout    time.Duration
)

func init() {
	flag.BoolVar(&serverMode, "s", false, "run in server mode")
	flag.StringVar(&serverAddr, "a", "127.0.0.1", "server IP address")
	flag.StringVar(&serverPort, "p", "10001", "server port")
	flag.StringVar(&localID, "l", "Alice", "local side ID")
	flag.StringVar(&remoteID, "r", "Bob", "remote side ID")
	flag.DurationVar(&timeout, "t", time.Second*2, "set reconnection timeout")
	flag.Parse()
}

func main() {

	if serverMode {
		logger, _ := zap.NewProduction()
		// create server
		s := server.NewServer(logger)
		fmt.Printf("\nserving at %v:%v\n", serverAddr, serverPort)
		// start listening to clients
		s.RunEchoServer(serverPort)
	} else {
		// create client
		c := client.NewClient(timeout, serverAddr, serverPort, localID, remoteID)
		fmt.Printf("\nconnecting to %v as %v via %v:%v\n",
			remoteID, localID, serverAddr, serverPort)

		// start sending echo messages
		echo(c)
	}
}

func echo(c *client.Client) {
	r, err := c.REconnect()
	if err != nil {
		fmt.Println("can't connect", err)
	}

	buf := make([]byte, 1024)
	for {
		// send echo to a remote client
		<-time.After(time.Second * 2)
		if _, err := c.Conn.WriteToUDP([]byte(fmt.Sprintf("echo from %v", localID)), r); err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("echo sent to %v\n", remoteID)
		}

		// read from connection until timeout
		// then send reconnect signal to restore p2p connection
		c.Conn.SetReadDeadline(time.Now().Add(time.Second * 3))
		n, err := c.Conn.Read(buf)
		if err != nil {
			fmt.Println(err)
			// on a successful REconnect a new session will be established and
			// the remote client port and address will be updated
			r, err = c.REconnect()
			if err != nil {
				fmt.Println(err)
				break
			}
			continue
		}
		fmt.Printf("received %v\n", string(buf[:n]))
	}
}
