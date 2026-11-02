package wgkeys

import (
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type KeyPair struct {
	PrivateKey string
	PublicKey  string
}

func Generate() (KeyPair, error) {
	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return KeyPair{}, err
	}
	return KeyPair{
		PrivateKey: key.String(),
		PublicKey:  key.PublicKey().String(),
	}, nil
}
