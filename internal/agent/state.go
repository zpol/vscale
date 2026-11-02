package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type State struct {
	ServerURL  string `json:"serverUrl"`
	AgentToken string `json:"agentToken"`
	DeviceID   uint   `json:"deviceId,omitempty"`
	HubID      uint   `json:"hubId,omitempty"`
	VscaleID   uint   `json:"vscaleId,omitempty"`
	Address    string `json:"address,omitempty"`
	PrivateKey string `json:"privateKey,omitempty"`
	PublicKey  string `json:"publicKey,omitempty"`
	Role       string `json:"role"` // device | hub
	Up         bool   `json:"up"`
}

func DefaultStatePath() string {
	if p := os.Getenv("VSCALE_STATE"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "./vscale-state.json"
	}
	return filepath.Join(home, ".config", "vscale", "state.json")
}

func LoadState(path string) (*State, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func SaveState(path string, s *State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}
