package diogenes

import (
	vault "github.com/hashicorp/vault/api"
	"log"
)

func (v *Vault) SetOnetimeToken(token string) {
	log.Printf("setting token to: %s", token)
	v.Connection.SetToken(token)
	log.Print("one time token set")
}

func (v *Vault) GetCurrentToken() string {
	return v.Connection.Token()
}

func (v *Vault) CreateOneTimeToken(policy []string) (string, error) {
	renew := false

	tokenRequest := vault.TokenCreateRequest{
		Policies:    policy,
		TTL:         "5m",
		DisplayName: "solonCreated",
		NumUses:     1,
		Renewable:   &renew,
	}

	log.Print("request created")

	resp, err := v.Connection.Auth().Token().Create(&tokenRequest)
	if err != nil {
		return "", err
	}

	return resp.Auth.ClientToken, nil
}
