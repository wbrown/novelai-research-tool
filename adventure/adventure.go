package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"unicode"

	"github.com/chzyer/readline"
	"github.com/jdkato/prose/v2"
	"github.com/wbrown/gpt_bpe"
	novelai_api "github.com/wbrown/novelai-research-tool/novelai-api"
)

//go:embed adventure.json
//go:embed adventure.txt

var f embed.FS

type Adventure struct {
	Parameters novelai_api.NaiGenerateParams
	Context    string
	API        novelai_api.NovelAiAPI
	Encoder    gpt_bpe.GPTEncoder
	MaxTokens  uint
}

func NewAdventure() (adventure Adventure) {
	parametersFile, _ := f.ReadFile("adventure.json")
	var parameters novelai_api.NaiGenerateParams
	if err := json.Unmarshal(parametersFile, &parameters); err != nil {
		log.Fatal(err)
	}
	contextBytes, _ := f.ReadFile("adventure.txt")
	adventure.Context = string(contextBytes)
	adventure.Parameters = parameters
	adventure.API = novelai_api.NewNovelAiAPI()
	adventure.Encoder = gpt_bpe.NewEncoder()
	adventure.MaxTokens = 1024 - *parameters.MaxLength
	return adventure
}

func (adventure Adventure) start() {
	fmt.Println(adventure.Context)
	adventure.Context = "[Narrative: second-person]\n" + adventure.Context
	rl, err := readline.New("> ")
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	var output string
	log.SetOutput(rl.Stderr())
	for {
		rl.ResetHistory()
		line, err := rl.Readline()
		if err != nil { // io.EOF
			break
		}
		adventure.Context = adventure.Context + "\n> " + line + "\n"
		for {
			tokens := adventure.Encoder.Encode(&adventure.Context)
			if uint(len(*tokens)) > adventure.MaxTokens {
				adventure.Context = strings.Join(
					strings.Split(adventure.Context, "\n")[1:], "\n")
			} else {
				break
			}
		}

		resp := adventure.API.GenerateWithParams(&adventure.Context,
			adventure.Parameters)
		output = resp.Response
		doc, err := prose.NewDocument(output)
		if err != nil {
			log.Fatal(err)
		}
		processed := make([]string, 0)

		for _, sent := range doc.Sentences() {
			lastChar := sent.Text[len(sent.Text)-1:]
			lastCharRune := []rune(lastChar)
			if unicode.IsPunct(lastCharRune[0]) {
				processed = append(processed, sent.Text)
			}
		}
		output = strings.Join(processed, " ")
		adventure.Context = adventure.Context + output
		fmt.Println(output)
	}
}

func main() {
	adventure := NewAdventure()
	adventure.start()
}
