package aimodules

import (
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/nacl/secretbox"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

type AIModule struct {
	Version       uint   `json:"moduleVersion"`
	EncodedData   string `json:"data"`
	EncryptedData []byte
	PrefixID      string
	Hash          string
	Name          string `json:"name"`
	Description   string `json:"description"`
	Model         string `json:"model"`
	Steps         uint   `json:"steps"`
}

func blake2bHash(size int, plaintext []byte) []byte {
	b2Hasher, _ := blake2b.New(size, nil)
	b2Hasher.Write(plaintext)
	return b2Hasher.Sum(nil)
}

func encryptPrefix(base64 string) (encrypted []byte, prefixId string, hash string) {
	var clearHash [32]byte
	var nonceHash [24]byte
	base64bytes := []byte(base64)
	clearHashSlice := blake2bHash(32, base64bytes)
	nonceHashSlice := blake2bHash(24, clearHashSlice)
	copy(clearHash[:], clearHashSlice[:32])
	copy(nonceHash[:], nonceHashSlice[:24])
	sealed := secretbox.Seal(nil, base64bytes, &nonceHash, &clearHash)
	encrypted = append(nonceHashSlice, sealed...)
	prefixHash := blake2bHash(32, encrypted)
	prefixId = fmt.Sprintf("%x", prefixHash)
	hash = fmt.Sprintf("%x", clearHash)

	return encrypted, prefixId, hash
}

func (aimodule *AIModule) ToPrefix() string {
	return fmt.Sprintf("%s:%s:%s", aimodule.Model,
		aimodule.PrefixID, aimodule.Hash)
}

func AIModuleFromArgs(id string, name string, description string) AIModule {
	idSplit := strings.Split(id, ":")
	return AIModule{
		Model:       idSplit[0],
		PrefixID:    idSplit[1],
		Hash:        idSplit[2],
		Name:        name,
		Description: description,
	}
}

func AIModuleFromFile(path string) AIModule {
	var aiModule AIModule
	configBytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("nrt: Error loading AI Module file `%s`: %v", path, err)
		os.Exit(1)
	}
	if err = json.Unmarshal(configBytes, &aiModule); err != nil {
		log.Printf("nrt: Error deserializing AI Module file `%s`: %v", path, err)
		os.Exit(1)
	}
	aiModule.EncryptedData, aiModule.PrefixID,
		aiModule.Hash = encryptPrefix(aiModule.EncodedData)
	return aiModule
}