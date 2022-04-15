package novelai_api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/kelseyhightower/envconfig"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/blake2b"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
)

type AuthConfig struct {
	Username   string `envconfig:"NAI_USERNAME"`
	Password   string `envconfig:"NAI_PASSWORD"`
	BackendURI string `envconfig:"NAI_BACKEND"`
}

type NaiKeys struct {
	EncryptionKey []byte
	AccessKey     string
	AccessToken   string
	Backend       string
}

func getAccessToken(access_key string, backendURI string) (accessToken string) {
	cl := http.DefaultClient
	params := make(map[string]string)
	params["key"] = access_key
	encoded, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", backendURI+"/user/login",
		bytes.NewBuffer(encoded))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent",
		"nrt/0.1 ("+runtime.GOOS+"; "+runtime.GOARCH+")")
	resp, err := cl.Do(req)
	if err != nil {
		log.Printf("auth: Error performing HTTP request: %v", err)
		return accessToken
	} else {
		resp_decoded := make(map[string]string)
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("auth: Error reading HTTP response body: %v", err)
			return accessToken
		}
		err = json.Unmarshal(body, &resp_decoded)
		if err != nil {
			log.Printf("body: %v", string(body))
			log.Printf("auth: Error unmarshaling JSON response: %v", err)
			return accessToken
		}
		accessToken = resp_decoded["accessToken"]
	}
	return accessToken
}

func naiHashArgon(size int, plaintext string, secret string, domain string) []byte {
	encoder, _ := blake2b.New(16, nil)
	encoder.Write([]byte(secret + domain))
	salt := encoder.Sum(nil)
	argon_key := argon2.IDKey([]byte(plaintext),
		salt,
		2,
		2000000/1024,
		1,
		uint32(size))
	return argon_key
}

func naiGenerateKeys(email string, password string) NaiKeys {
	pw_email_secret := password[0:6] + email
	encryption_key := naiHashArgon(128,
		password,
		pw_email_secret,
		"novelai_data_encryption_key")
	access_key := naiHashArgon(64,
		password,
		pw_email_secret,
		"novelai_data_access_key")[0:64]
	return NaiKeys{
		EncryptionKey: encryption_key,
		AccessKey: strings.Replace(
			strings.Replace(
				base64.StdEncoding.EncodeToString(access_key)[0:64],
				"/", "_", -1),
			"+", "-", -1),
	}
}

func generateUsernames(email string) (usernames []string) {
	usernames = append(usernames, strings.ToLower(email))
	if usernames[0] != email {
		usernames = append(usernames, email)
	}
	usernames = append(usernames, strings.ToTitle(email[0:1])+email[1:])
	return usernames
}

func Auth(email string, password string, backendURI string) (keys NaiKeys) {
	usernames := generateUsernames(email)
	for userIdx := range usernames {
		username := usernames[userIdx]
		keys = naiGenerateKeys(username, password)
		log.Printf("auth: authenticating for '%s'\n", username)
		keys.AccessToken = getAccessToken(keys.AccessKey, backendURI)
		if len(keys.AccessToken) == 0 {
			log.Printf("auth: failed for '%s'\n", username)
		} else {
			break
		}
	}
	return keys
}

func AuthEnv() NaiKeys {
	var authCfg AuthConfig
	err := envconfig.Process("", &authCfg)
	if err != nil {
		log.Printf("auth: Error processing environment: %v", err)
		os.Exit(1)
	}
	if len(authCfg.Username) == 0 || len(authCfg.Password) == 0 {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n",
			"Please ensure that NAI_USERNAME and NAI_PASSWORD are set in your environment.")
		os.Exit(1)
	}
	if len(authCfg.BackendURI) == 0 {
		authCfg.BackendURI = "https://api.novelai.net"
	} else {
		authCfg.BackendURI = strings.TrimSuffix(authCfg.BackendURI, "/")
	}
	auth := Auth(authCfg.Username, authCfg.Password, authCfg.BackendURI)
	auth.Backend = authCfg.BackendURI
	if len(auth.AccessToken) == 0 {
		log.Printf("auth: failed to obtain AccessToken!")
		os.Exit(1)
	}
	return auth
}
