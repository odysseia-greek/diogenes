package diogenes

import (
	"context"
	"fmt"
	"github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/kubernetes"
	"github.com/hashicorp/vault/http"
	"github.com/hashicorp/vault/vault"
	"log"
	"strings"
	"testing"
	"time"
)

type Client interface {
	CheckHealthyStatus(ticks, tick time.Duration) bool
	Health() (bool, error)
	CreateOneTimeToken(policy []string) (string, error)
	CreateNewSecret(name string, payload []byte) (bool, error)
	GetSecret(name string) (*api.Secret, error)
	SetOnetimeToken(token string)
	GetCurrentToken() string
}

type Vault struct {
	SecretPath string
	Connection *api.Client
}

const (
	defaultPath       string = "configs/data"
	fixtureSecretName string = "isitsecretisitsafe"
	fixtureKey        string = "keytothesidedoor"
	fixtureValue      string = "oferebor"
)

func NewVaultClient(address, token string, tlsConfig *api.TLSConfig) (Client, error) {
	//https://patorjk.com/software/taag/#p=display&f=Crawford2&t=DIOGENES
	log.Print("\n ___    ____  ___    ____    ___  ____     ___  _____\n|   \\  |    |/   \\  /    |  /  _]|    \\   /  _]/ ___/\n|    \\  |  ||     ||   __| /  [_ |  _  | /  [_(   \\_ \n|  D  | |  ||  O  ||  |  ||    _]|  |  ||    _]\\__  |\n|     | |  ||     ||  |_ ||   [_ |  |  ||   [_ /  \\ |\n|     | |  ||     ||     ||     ||  |  ||     |\\    |\n|_____||____|\\___/ |___,_||_____||__|__||_____| \\___|\n                                                     \n")
	log.Print(strings.Repeat("~", 37))
	log.Print("\"ἄνθρωπον ζητῶ\"")
	log.Print("\"I am looking for an honest man.\"")
	log.Print(strings.Repeat("~", 37))

	config := api.Config{
		Address: address,
	}

	log.Print(tlsConfig)

	if tlsConfig != nil {
		log.Print(fmt.Sprintf("TLS config found setting to: %s", tlsConfig.CAPath))
		err := config.ConfigureTLS(tlsConfig)
		if err != nil {
			return nil, err
		}
	}

	client, err := api.NewClient(&config)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize Vault client: %w", err)
	}

	log.Print("created new vault client")

	if token != "" {
		log.Printf("setting roottoken to: %s", token)
		client.SetToken(token)
	} else {
		log.Print("no token set")
	}

	return &Vault{Connection: client, SecretPath: defaultPath}, nil
}

func NewMockVaultClient(t *testing.T) (Client, error) {
	t.Helper()

	core, keyShares, rootToken := vault.TestCoreUnsealed(t)
	_ = keyShares

	ln, addr := http.TestServer(t, core)

	defer ln.Close()
	conf := api.DefaultConfig()
	conf.Address = addr

	client, err := api.NewClient(conf)
	if err != nil {
		return nil, err
	}
	client.SetToken(rootToken)

	if err != nil {
		t.Fatal(err)
	}

	mount := api.MountInput{
		Type:                  "kv",
		Description:           "",
		Config:                api.MountConfigInput{},
		Local:                 false,
		SealWrap:              false,
		ExternalEntropyAccess: false,
		Options:               nil,
		PluginName:            "",
	}

	err = client.Sys().Mount(defaultPath, &mount)

	fixtureSecret := fmt.Sprintf("%s/%s", defaultPath, fixtureSecretName)
	_, err = client.Logical().Write(fixtureSecret, map[string]interface{}{
		fixtureKey: fixtureValue,
	})

	return &Vault{Connection: client, SecretPath: defaultPath}, nil
}

func CreateVaultClientKubernetes(address, vaultRole, jwt string, tlsConfig *api.TLSConfig) (Client, error) {
	config := api.Config{
		Address: address,
	}

	log.Print(tlsConfig)

	if tlsConfig != nil {
		log.Print(fmt.Sprintf("TLS config found setting to: %s", tlsConfig.CAPath))
		err := config.ConfigureTLS(tlsConfig)
		if err != nil {
			return nil, err
		}
	}

	client, err := api.NewClient(&config)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize Vault client: %w", err)
	}

	k8sAuth, err := auth.NewKubernetesAuth(
		vaultRole,
		auth.WithServiceAccountToken(jwt),
	)

	// log in to Vault's Kubernetes auth method
	resp, err := client.Auth().Login(context.TODO(), k8sAuth)
	if err != nil {
		return nil, fmt.Errorf("unable to log in with Kubernetes auth: %w", err)
	}
	if resp == nil || resp.Auth == nil || resp.Auth.ClientToken == "" {
		return nil, fmt.Errorf("login response did not return client token")
	}

	client.SetToken(resp.Auth.ClientToken)

	return &Vault{Connection: client, SecretPath: defaultPath}, nil
}

func CreateTLSConfig(insecure bool, ca, cert, key, caPath string) *api.TLSConfig {
	return &api.TLSConfig{
		CAPath:     caPath,
		CACert:     ca,
		ClientCert: cert,
		ClientKey:  key,
		Insecure:   insecure,
	}
}
