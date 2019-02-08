package server

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/faroyam/udp-hole-punch/utils"

	"net"
	"sync"

	"go.uber.org/zap"
)

// Server implementation
type Server struct {
	sync.RWMutex
	m map[string]utils.ClientData
	l *zap.Logger
}

// addClient adds cleint to the map
func (s *Server) addClient(c utils.ClientData) {
	s.Lock()
	defer s.Unlock()
	s.m[c.LocalID] = c
}

// checkClient checks if client already in the map
func (s *Server) checkClient(ID string) (utils.ClientData, error) {
	s.RLock()
	defer s.RUnlock()
	i, ok := s.m[ID]
	if ok {
		return i, nil
	}
	return i, errors.New("no such client")
}

// deleteClient deletes client from the map
func (s *Server) deleteClient(ID string) {
	s.RLock()
	defer s.RUnlock()
	delete(s.m, ID)
}

// RunEchoServer starts listening socket
func (s *Server) RunEchoServer(port string) error {

	serverAddr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		s.l.Warn("Error", zap.Error(err))
		return err
	}

	serverConn, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		s.l.Warn("Error", zap.Error(err))
		return err
	}

	defer serverConn.Close()

	for {
		buf := make([]byte, 1024)
		n, newClientAddr, err := serverConn.ReadFromUDP(buf)
		if err != nil {
			s.l.Warn("Error", zap.Error(err))
			continue
		}

		// handle each new client in a new goroutine
		go func() {
			incomingRequest := utils.ConnectRequest{}
			json.Unmarshal(buf[:n], &incomingRequest)

			s.l.Info("client",
				zap.String("from", incomingRequest.LocalID),
				zap.String("to", incomingRequest.RemoteID),
			)

			newClientIP := newClientAddr.IP.String()
			newClientPort := strconv.Itoa(newClientAddr.Port)

			clientFromMap, err := s.checkClient(incomingRequest.RemoteID)
			// add a new client to the map if requested client is not found in the map
			// or if it is waiting for a client with another ID
			if err != nil || clientFromMap.RemoteID != incomingRequest.LocalID {
				s.addClient(utils.ClientData{
					RemoteID:  incomingRequest.RemoteID,
					LocalID:   incomingRequest.LocalID,
					LocalIP:   newClientIP,
					LocalPort: newClientPort,
				})

				s.l.Info("client added to waiting list",
					zap.Error(err), zap.String("addr", newClientIP),
					zap.String("port", newClientPort),
				)
				return
			}

			responseToClientFromMap := utils.ConnectResponse{
				RemoteIP:   clientFromMap.LocalIP,
				RemotePort: clientFromMap.LocalPort,
			}

			responseToClientFromMapJSON, err := json.Marshal(responseToClientFromMap)
			if err != nil {
				s.l.Warn("Error", zap.Error(err))
				return
			}

			responseToNewClient := utils.ConnectResponse{
				RemoteIP:   newClientIP,
				RemotePort: newClientPort,
			}

			responseToNewClientJSON, err := json.Marshal(responseToNewClient)
			if err != nil {
				s.l.Warn("Error", zap.Error(err))
				return
			}

			_, err = serverConn.WriteToUDP(responseToClientFromMapJSON, newClientAddr)
			if err != nil {
				s.l.Warn("Error", zap.Error(err))
				return
			}

			clientFromMapUDPAddr, _ := net.ResolveUDPAddr("udp",
				clientFromMap.LocalIP+":"+clientFromMap.LocalPort)

			_, err = serverConn.WriteToUDP(responseToNewClientJSON, clientFromMapUDPAddr)
			if err != nil {
				s.l.Warn("Error", zap.Error(err))
				return
			}

			s.deleteClient(clientFromMap.LocalID)

			s.l.Info("clients connected",
				zap.String("client", incomingRequest.LocalID),
				zap.String("client", incomingRequest.RemoteID),
			)

		}()
	}
}

// NewServer creates a server instance with a logger and with an empty map
func NewServer(l *zap.Logger) *Server {
	return &Server{m: make(map[string]utils.ClientData), l: l}
}
