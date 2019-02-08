package utils

// ConnectRequest decribes message from client to server, contents requested and local clients ID
type ConnectRequest struct {
	LocalID  string `json:"local_id"`
	RemoteID string `json:"remote_id"`
}

// ConnectResponse decribes message from server to client, contents requested client address
type ConnectResponse struct {
	RemoteIP   string `json:"local_addr"`
	RemotePort string `json:"remote_port"`
}

// ClientData contents all data about client to be stored on the server
type ClientData struct {
	RemoteID  string `json:"remote_addr"`
	LocalIP   string `json:"local_addr"`
	LocalPort string `json:"local_port"`
	LocalID   string `json:"local_ID"`
}
