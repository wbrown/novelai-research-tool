package main

import (
	"encoding/json"
	"fmt"
	novelai_api "github.com/wbrown/novelai-research-tool/novelai-api"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

type ContentTest struct {
	OutputFilename string `json:"output_filename"`
	PromptFilename string `json:"prompt_filename"`
	Iterations int `json:"iterations"`
	Generations int `json:"generations"`
	Parameters novelai_api.NaiGenerateParams `json:"parameters"`
	Prompt string
	API novelai_api.NovelAiAPI
}

type IterationResult struct {
	Parameters novelai_api.NaiGenerateParams `json:"settings"`
	Prompt string `json:"prompt"`
	Result string `json:"result"`
	Responses []string `json:"responses"`
}

func (ct ContentTest) performGenerations(generations int, input string) (responses []string) {
	context := input
	for generation := 0; generation < generations; generation++ {
		resp := ct.API.GenerateWithParams(context, ct.Parameters)
		responses = append(responses, resp)
		fmt.Printf("=> %v\n", resp)
		context = context + resp
		time.Sleep(1 * time.Second)
	}
	return responses
}

func (ct ContentTest) Perform() {
	promptBytes, err := ioutil.ReadFile(ct.PromptFilename)
	if err != nil {
		log.Fatal(err)
	}
	ct.Prompt = string(promptBytes)
	for iteration := 0; iteration < ct.Iterations; iteration++ {
		responses := ct.performGenerations(ct.Generations, ct.Prompt)
		f, err := os.OpenFile(ct.OutputFilename,
			os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatal(err)
		}
		serialized, err := json.MarshalIndent(IterationResult{
			ct.Parameters,
			ct.Prompt,
			strings.Join(responses, ""),
			responses,
		}, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		_, err = f.Write(serialized)
		if err != nil {
			log.Fatal(err)
		}
		_, err = f.WriteString("\n")
		if err != nil {
			log.Fatal(err)
		}
		f.Close()
	}
}

func LoadTestFromFile(path string) (test ContentTest) {
	configBytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(configBytes, &test)
	if err != nil {
		log.Fatal(err)
	}
	return test
}

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("(NovelAI Research Tool) %s dir/test.json\n", os.Args[0])
		os.Exit(1)
	}
	inputPath := os.Args[1]
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Printf("`%v` does not exist!\n", inputPath)
		os.Exit(1)
	}
	test := LoadTestFromFile(inputPath)
	test.API = novelai_api.NewNovelAiAPI()
	test.Perform()
}