package main

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	novelai_api "github.com/wbrown/novelai-research-tool/novelai-api"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ContentTest struct {
	OutputPrefix string `json:"output_prefix"`
	PromptFilename string `json:"prompt_filename"`
	Iterations int `json:"iterations"`
	Generations int `json:"generations"`
	Parameters novelai_api.NaiGenerateParams `json:"parameters"`
	Prompt string
	WorkingDir string
	OutputPath string
	PromptPath string
	API novelai_api.NovelAiAPI
}

type IterationResult struct {
	Parameters novelai_api.NaiGenerateParams `json:"settings"`
	Prompt string `json:"prompt"`
	Result string `json:"result"`
	Responses []string `json:"responses"`
}

func (ct ContentTest) performGenerations(generations int, input string) (responses []string) {
	genResMarker := color.New(color.FgWhite, color.BgGreen).SprintFunc()
	newLineToken := genResMarker("\\n")+"\n"
	context := input
	for generation := 0; generation < generations; generation++ {
		resp := ct.API.GenerateWithParams(context, ct.Parameters)
		responses = append(responses, resp)
		fmt.Printf("%v%v\n", genResMarker("=>"),
			strings.Replace(resp,"\n", newLineToken, -1))
		context = context + resp
		time.Sleep(1 * time.Second)
	}
	return responses
}

func handleWrite(f *os.File, s string) {
	_, err := f.WriteString(s)
	if err != nil {
		log.Fatal(err)
	}
}

func (ct ContentTest) Perform() {
	promptMarker := color.New(color.FgWhite, color.BgBlue).SprintFunc()
	newLineToken := promptMarker("\\n")+"\n"
	promptBytes, err := ioutil.ReadFile(ct.PromptPath)
	if err != nil {
		log.Fatal(err)
	}
	ct.Prompt = string(promptBytes)
	f, err := os.OpenFile(ct.OutputPath,
		os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	handleWrite(f, "[")
	for iteration := 0; iteration < ct.Iterations; iteration++ {
		if iteration != 0 {
			handleWrite(f, ",\n")
		}
		fmt.Printf("%v%v\n", promptMarker("<="),
			strings.Replace(ct.Prompt, "\n", newLineToken, -1))
		responses := ct.performGenerations(ct.Generations, ct.Prompt)
		serialized, err := json.MarshalIndent(IterationResult{
			ct.Parameters,
			ct.Prompt,
			strings.Join(responses, ""),
			responses,
		}, " ", "  ")
		if err != nil {
			log.Fatal(err)
		}
		handleWrite(f, string(serialized))
		f.Sync()
	}
	handleWrite(f, "]")
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
	test.WorkingDir = filepath.Dir(path)
	test.PromptPath = filepath.Join(test.WorkingDir, test.PromptFilename)
	if _, err := os.Stat(test.PromptPath); os.IsNotExist(err) {
		fmt.Printf("`%v` does not exist!\n", test.PromptPath)
		os.Exit(1)
	}
	test.OutputPath = filepath.Join(test.WorkingDir,
		strings.Join([]string{
		test.OutputPrefix,
		test.Parameters.Model,
		test.Parameters.Prefix,
		time.Now().Format("2006-01-02T150405Z0700") + ".json"}, "-"))
	return test
}

func main() {
	binName := filepath.Base(os.Args[0])

	if len(os.Args) != 2 {
		fmt.Printf("%v: %s dir/test.json\n", binName, os.Args[0])
		os.Exit(1)
	}
	inputPath := os.Args[1]
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Printf("%v: `%v` does not exist!\n", binName, inputPath)
		os.Exit(1)
	}
	test := LoadTestFromFile(inputPath)
	test.API = novelai_api.NewNovelAiAPI()
	test.Perform()
}