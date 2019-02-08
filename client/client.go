package client

import (
	"encoding/json"
	"errors"
	"net"
	"time"

	"github.com/faroyam/udp-hole-punch/utils"
)

// Client implementation
type Client struct {
	connectStatus chan error
	stopChan      chan struct{}

	Conn *net.UDPConn
	R    *utils.ConnectResponse

	serverAddr string
	serverPort string
	localID    string
	remoteID   string

	timeout time.Duration
}

// NewClient returns new client
func NewClient(timeout time.Duration, serverAddr, serverPort, localID, remoteID string) *Client {
	client := &Client{
		connectStatus: make(chan error),
		stopChan:      make(chan struct{}),
		timeout:       timeout,
		serverAddr:    serverAddr,
		serverPort:    serverPort,
		localID:       localID,
		remoteID:      remoteID,
	}
	return client
}

// createP2PConnection sends local and remote ID to the server and waits for the response
func (c *Client) createP2PConnection() {
	var err error
	defer func() { c.connectStatus <- err }()

	response := utils.ConnectResponse{}
	request := utils.ConnectRequest{LocalID: c.localID, RemoteID: c.remoteID}
	buf := make([]byte, 1024)

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return
	}

	serverUDPAddr, err := net.ResolveUDPAddr("udp",
		c.serverAddr+":"+c.serverPort)
	if err != nil {
		return
	}

	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return
	}
L:
	for {
		select {
		case <-c.stopChan:
			return
		default:
			conn.SetReadDeadline(time.Now().Add(c.timeout + c.timeout/2))
			_, err := conn.WriteToUDP(requestJSON, serverUDPAddr)
			if err != nil {
				continue
			}

			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				continue
			}
			err = json.Unmarshal(buf[:n], &response)
			if err != nil {
				continue
			}
			conn.SetReadDeadline(time.Now().Add(c.timeout + c.timeout/2))
			break L
		}
	}

	c.Conn = conn
	c.R = &response
	return
}

// REconnect allows to manually send reconnect signal, returns new remote addr and port
func (c *Client) REconnect() (*net.UDPAddr, error) {
	go c.createP2PConnection()

	select {
	case err := <-c.connectStatus:
		if err != nil {
			return nil, err
		}
		remoteUDP, err := net.ResolveUDPAddr("udp", c.R.RemoteIP+":"+c.R.RemotePort)
		return remoteUDP, err

	case <-time.After(time.Second * 60):
		c.stopChan <- struct{}{}
		return nil, errors.New("reconnection timeout")
	}
}
