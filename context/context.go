package context

import (
	"encoding/json"
	"fmt"
	gpt_bpe "github.com/wbrown/novelai-research-tool/gpt-bpe"
	novelai_api "github.com/wbrown/novelai-research-tool/novelai-api"
	"log"
	"os"
)

type SimpleContext struct {
	Parameters  novelai_api.NaiGenerateParams
	Context     string
	Memory      string
	FullReturn  bool
	AuthorsNote string
	LastContext string
	API         novelai_api.NovelAiAPI
	Encoder     gpt_bpe.GPTEncoder
	MaxTokens   uint
}

func NewSimpleContext() (context SimpleContext) {
	parametersFile, _ := os.ReadFile("params.json")
	var parameters novelai_api.NaiGenerateParams
	if err := json.Unmarshal(parametersFile, &parameters); err != nil {
		println("Error in JSON.")
		fmt.Scanln()
		log.Fatal(err)
	}
	contextBytes, _ := os.ReadFile("prompt.txt")
	context.Context = string(contextBytes)
	context.LastContext = string(contextBytes)
	context.Parameters = parameters
	context.API = novelai_api.NewNovelAiAPI()
	context.Encoder = gpt_bpe.NewEncoder()
	context.FullReturn = parameters.ReturnFullText
	context.MaxTokens = 2048 - *parameters.MaxLength
	context.Memory = parameters.LogitMemory
	context.AuthorsNote = parameters.LogitAuthors
	if context.Parameters.Prefix != "vanilla" {
		context.MaxTokens = context.MaxTokens - 20
	}

	return context
}

func (ctx SimpleContext) SaveContext(path string) {
	if f, err := os.Create(path); err != nil {
		println("\n\n\n\nError saving file.")
		log.Fatal(err)
	} else if _, err = f.WriteString(ctx.Context); err != nil {
		println("\n\n\n\nError saving file.")
		log.Fatal(err)
	}
}