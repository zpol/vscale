package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type APIClient struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func NewAPIClient(base, token string) *APIClient {
	return &APIClient{
		BaseURL: stringsTrimRight(base, "/"),
		Token:   token,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

func stringsTrimRight(s, cut string) string {
	for len(s) > 0 && s[len(s)-1:] == cut {
		s = s[:len(s)-1]
	}
	return s
}

type EnrollRequest struct {
	Token    string `json:"token"`
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
}

type EnrollResponse struct {
	DeviceID   uint   `json:"deviceId"`
	VscaleID   uint   `json:"vscaleId"`
	AgentToken string `json:"agentToken"`
	Address    string `json:"address"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
	ServerURL  string `json:"serverUrl"`
}

type PeerConfig struct {
	PublicKey  string   `json:"publicKey"`
	Endpoint   string   `json:"endpoint,omitempty"`
	AllowedIPs []string `json:"allowedIps"`
	Keepalive  int      `json:"persistentKeepalive,omitempty"`
}

type ConfigResponse struct {
	Role        string       `json:"role"`
	Interface   string       `json:"interface"`
	Address     string       `json:"address"`
	PrivateKey  string       `json:"privateKey"`
	ListenPort  int          `json:"listenPort,omitempty"`
	DNS         string       `json:"dns,omitempty"`
	MTU         int          `json:"mtu"`
	Peers       []PeerConfig `json:"peers"`
	Routes      []string     `json:"routes"`
	VscaleID    uint         `json:"vscaleId"`
	OverlayCIDR string       `json:"overlayCidr"`
}

type HeartbeatRequest struct {
	TransferRx int64      `json:"transferRx"`
	TransferTx int64      `json:"transferTx"`
	LocalCIDRs []string   `json:"localCidrs"`
	Hostname   string     `json:"hostname"`
	OS         string     `json:"os"`
	PeerStats  []PeerStat `json:"peerStats"`
}

type PeerStat struct {
	PublicKey     string `json:"publicKey"`
	TransferRx    int64  `json:"transferRx"`
	TransferTx    int64  `json:"transferTx"`
	LastHandshake int64  `json:"lastHandshake"`
}

func (c *APIClient) do(method, path string, body any, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("X-Vscale-Token", c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("api %s: %s", resp.Status, string(data))
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}

func (c *APIClient) Enroll(req EnrollRequest) (*EnrollResponse, error) {
	var out EnrollResponse
	if err := c.do(http.MethodPost, "/api/agent/enroll", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *APIClient) Config() (*ConfigResponse, error) {
	var out ConfigResponse
	if err := c.do(http.MethodGet, "/api/agent/config", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *APIClient) Heartbeat(req HeartbeatRequest) error {
	return c.do(http.MethodPost, "/api/agent/heartbeat", req, nil)
}
