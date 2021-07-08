package novelai_api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/kelseyhightower/envconfig"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/blake2b"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type AuthConfig struct {
	Username string `envconfig:"NAI_USERNAME"`
	Password string `envconfig:"NAI_PASSWORD"`
}

type NaiKeys struct {
	EncryptionKey []byte
	AccessKey string
	AccessToken string
}

func getAccessToken(access_key string) string {
	cl := http.DefaultClient
	params := make(map[string]string)
	params["key"] = access_key
	encoded, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", "https://api.novelai.net/user/login",
		bytes.NewBuffer(encoded))
	req.Header.Set("Content-Type", "application/json")
	resp, err := cl.Do(req)
	if err != nil {
		log.Fatal(err)
	} else {
		resp_decoded := make(map[string]string)
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		err = json.Unmarshal(body, &resp_decoded)
		return resp_decoded["accessToken"]
	}
	return ""
}

func naiHashArgon(size int, plaintext string, secret string, domain string) []byte {
	encoder, _ := blake2b.New(16, nil)
	encoder.Write([]byte(secret+domain))
	salt := encoder.Sum(nil)
	argon_key := argon2.IDKey([]byte(plaintext),
		salt,
		2,
		2000000 / 1024,
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

func Auth(email string, password string) NaiKeys {
	keys := naiGenerateKeys(email, password)
	keys.AccessToken = getAccessToken(keys.AccessKey)
	return keys
}

func AuthEnv() NaiKeys {
	var authCfg AuthConfig
	err := envconfig.Process("", &authCfg)
	if err != nil {
		log.Fatal(err)
	}
	return Auth(authCfg.Username, authCfg.Password)
}