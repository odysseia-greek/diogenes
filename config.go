package diogenes

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/vault/api"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

const (
	defaultVault            = "https://vault:8200"
	VAULT                   = "vault"
	defaultNamespace        = "odysseia"
	defaultRoleName         = "solon"
	EnvVaultService         = "VAULT_SERVICE"
	EnvRootToken            = "VAULT_ROOT_TOKEN"
	EnvAuthMethod           = "AUTH_METHOD"
	EnvTLSEnabled           = "VAULT_TLS"
	EnvVaultRole            = "VAULT_ROLE"
	EnvRootTlSDir           = "CERT_ROOT"
	AuthMethodKube          = "kubernetes"
	AuthMethodToken         = "token"
	defaultTLSFileLocation  = "/etc/certs"
	serviceAccountTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

func getStringFromEnv(envName, defaultValue string) string {
	var value string
	value = os.Getenv(envName)
	if value == "" {
		log.Printf("%s empty set as env variable - defaulting to %s", envName, defaultValue)
		value = defaultValue
	}

	return value
}

func getBoolFromEnv(envName string) bool {
	var value bool
	envValue := os.Getenv(envName)
	if envValue == "" || envValue == "no" || envValue == "false" {
		value = false
	} else {
		value = true
	}

	return value
}

func CreateVaultClient(env string, healthCheck bool) (Client, error) {
	var vaultClient Client

	vaultRootToken := getStringFromEnv(EnvRootToken, "")
	vaultAuthMethod := getStringFromEnv(EnvAuthMethod, AuthMethodToken)
	vaultService := getStringFromEnv(EnvVaultService, defaultVault)
	vaultRole := getStringFromEnv(EnvVaultRole, defaultRoleName)
	tlsEnabled := getBoolFromEnv(EnvTLSEnabled)
	rootPath := getStringFromEnv(EnvRootTlSDir, defaultTLSFileLocation)
	secretPath := filepath.Join(rootPath, VAULT)

	log.Printf("vaultAuthMethod set to %s", vaultAuthMethod)
	log.Printf("secretPath set to %s", secretPath)
	log.Printf("tlsEnabled set to %v", tlsEnabled)

	var tlsConfig *api.TLSConfig

	if tlsEnabled {
		insecure := false
		if env == "LOCAL" || env == "TEST" {
			insecure = !insecure
			secretPath = "/tmp"
		}

		ca := fmt.Sprintf("%s/vault.ca", secretPath)
		cert := fmt.Sprintf("%s/vault.crt", secretPath)
		key := fmt.Sprintf("%s/vault.key", secretPath)

		log.Print(ca)
		log.Print(cert)
		log.Print(key)

		tlsConfig = CreateTLSConfig(insecure, ca, cert, key, secretPath)
	}

	if vaultAuthMethod == AuthMethodKube {
		jwt, err := os.ReadFile(serviceAccountTokenPath)
		if err != nil {
			log.Print(err)
			return nil, err
		}

		vaultToken := string(jwt)

		client, err := CreateVaultClientKubernetes(vaultService, vaultRole, vaultToken, tlsConfig)
		if err != nil {
			log.Print(err)
			return nil, err
		}

		if healthCheck {
			ticks := 120 * time.Second
			tick := 1 * time.Second
			healthy := vaultClient.CheckHealthyStatus(ticks, tick)
			if !healthy {
				return nil, fmt.Errorf("error getting healthy status from vault")
			}
		}

		vaultClient = client
	} else {
		if env == "LOCAL" || env == "TEST" {
			log.Print("local testing, getting token from file")
			localToken, err := getTokenFromFile(defaultNamespace)
			if err != nil {
				log.Print(err)
				return nil, err
			}
			client, err := NewVaultClient(vaultService, localToken, tlsConfig)
			if err != nil {
				log.Print(err)
				return nil, err
			}

			vaultClient = client
		} else {
			client, err := NewVaultClient(vaultService, vaultRootToken, tlsConfig)
			if err != nil {
				log.Print(err)
				return nil, err
			}

			vaultClient = client
		}
	}

	return vaultClient, nil
}

func getTokenFromFile(namespace string) (string, error) {
	clusterKeys := filepath.Join("eratosthenes", fmt.Sprintf("cluster-keys-%s.json", namespace))

	f, err := ioutil.ReadFile(clusterKeys)
	if err != nil {
		log.Print(fmt.Sprintf("Cannot read fixture file: %s", err))
		return "", err
	}

	var result map[string]interface{}

	// Unmarshal or Decode the JSON to the interface.
	json.Unmarshal(f, &result)

	return result["root_token"].(string), nil
}
