package main

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	gpt_bpe "github.com/wbrown/novelai-research-tool/gpt-bpe"
	novelai_api "github.com/wbrown/novelai-research-tool/novelai-api"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

type PermutationsSpec struct {
	Model                  []string     `json:"model"`
	Prefix                 []string     `json:"prefix"`
	Temperature            []float64    `json:"temperature"`
	MaxLength              []uint       `json:"max_length"`
	MinLength              []uint       `json:"min_length"`
	TopK                   []uint       `json:"top_k"`
	TopP                   []float64    `json:"top_p"`
	TailFreeSampling       []float64    `json:"tail_free_sampling"`
	RepetitionPenalty      []float64    `json:"repetition_penalty"`
	RepetitionPenaltyRange []uint       `json:"repetition_penalty_range"`
	RepetitionPenaltySlope []float64    `json:"repetition_penalty_slope"`
}

type ContentTest struct {
	OutputPrefix string `json:"output_prefix"`
	PromptFilename string `json:"prompt_filename"`
	Iterations int `json:"iterations"`
	Generations int `json:"generations"`
	Parameters novelai_api.NaiGenerateParams `json:"parameters"`
	Permutations PermutationsSpec `json:"permutations"`
	Prompt string
	WorkingDir string
	OutputPath string
	PromptPath string
	API novelai_api.NovelAiAPI
}

type EncodedIterationResult struct {
	Prompt string `json:"prompt""`
	Responses []string `json:"responses""`
}

type IterationResult struct {
	Parameters novelai_api.NaiGenerateParams `json:"settings"`
	Prompt string `json:"prompt"`
	Result string `json:"result"`
	Responses []string `json:"responses"`
	Encoded EncodedIterationResult `json:"encoded"`
}

func (ct ContentTest) performGenerations(generations int, input string) (results IterationResult) {
	genResMarker := color.New(color.FgWhite, color.BgGreen).SprintFunc()
	newLineToken := genResMarker("\\n")+"\n"
	context := input
	results.Prompt = input
	results.Parameters = ct.Parameters
	throttle := time.NewTimer(1100 * time.Millisecond)
	for generation := 0; generation < generations; generation++ {
		resp := ct.API.GenerateWithParams(context, ct.Parameters)
		if generation == 0 {
			results.Encoded.Prompt = resp.EncodedRequest
		}
		results.Responses = append(results.Responses, resp.Response)
		results.Encoded.Responses = append(results.Encoded.Responses, resp.EncodedResponse)
		fmt.Printf("%v%v\n", genResMarker("=>"),
			strings.Replace(resp.Response,"\n", newLineToken, -1))
		context = context + resp.Response
		<-throttle.C
		throttle = time.NewTimer(1100 * time.Millisecond)
	}
	results.Result = strings.Join(results.Responses, "")
	return results
}

func handleWrite(f *os.File, s string) {
	_, err := f.WriteString(s)
	if err != nil {
		log.Fatal(err)
	}
}

func (ct ContentTest) GeneratePermutations() (tests []ContentTest) {
	permutations := []novelai_api.NaiGenerateParams{ct.Parameters}
	// Loop over the fields in `Permutations` type.
	fields := reflect.TypeOf(ct.Permutations)
	for field := 0; field < fields.NumField(); field++ {
		// For each field, check the field contents and determine if there's any
		// values in there.
		fieldName := fields.FieldByIndex([]int{field}).Name
		fieldValues := reflect.ValueOf(ct.Permutations).Field(field)
		if fieldValues.Len() > 1 {
			// Loop over the values in the field to permute on.
			newPermutations := make([]novelai_api.NaiGenerateParams, 0)
			// Loop over each permutation we already have existing.
			for permutationTargetIdx := range permutations {
				// Create a new permutation for each value in the field.
				for valIdx := 0; valIdx < fieldValues.Len(); valIdx++ {
					value := fieldValues.Index(valIdx)
					permutation := permutations[permutationTargetIdx]
					targetField, _ := reflect.TypeOf(permutation).FieldByName(fieldName)
					reflect.ValueOf(&permutation).Elem().Field(targetField.Index[0]).Set(
						value)
					newPermutations = append(newPermutations, permutation)
				}
			}
			permutations = newPermutations
		}
	}
	for permutationIdx := range permutations {
		newTest := ct
		newTest.Parameters = permutations[permutationIdx]
		tests = append(tests, newTest)
	}
	return tests
}

func (ct ContentTest) Perform() {
	promptMarker := color.New(color.FgWhite, color.BgBlue).SprintFunc()
	newLineToken := promptMarker("\\n")+"\n"
	promptBytes, err := ioutil.ReadFile(ct.PromptPath)
	if err != nil {
		log.Fatal(err)
	}
	ct.OutputPath = filepath.Join(ct.WorkingDir,
		strings.Join([]string{
			ct.OutputPrefix,
			ct.Parameters.Model,
			ct.Parameters.Prefix,
			time.Now().Format("2006-01-02T150405Z0700") + ".json"}, "-"))
	ct.Prompt = string(promptBytes)
	f, err := os.OpenFile(ct.OutputPath,
		os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	handleWrite(f, "[")
	paramReport, _ := json.MarshalIndent(ct.Parameters, "             ", " ")
	fmt.Printf("%v %v\n", promptMarker("Parameters:"), string(paramReport))
	for iteration := 0; iteration < ct.Iterations; iteration++ {
		fmt.Printf("%v %v / %v\n", promptMarker("Iteration:"),
			iteration+1, ct.Iterations)
		if iteration != 0 {
			handleWrite(f, ",\n")
		}
		fmt.Printf("%v%v\n", promptMarker("<="),
			strings.Replace(ct.Prompt, "\n", newLineToken, -1))
		responses := ct.performGenerations(ct.Generations, ct.Prompt)
		serialized, err := json.MarshalIndent(responses, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		handleWrite(f, string(serialized))
		f.Sync()
	}
	handleWrite(f, "]")
}

func GenerateTestsFromFile(path string) (tests []ContentTest) {
	configBytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	var test ContentTest
	test.API = novelai_api.NewNovelAiAPI()
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
	return test.GeneratePermutations()
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
	encoder := gpt_bpe.NewEncoder()
	scenario := ScenarioFromFile(&encoder, inputPath)
	scenario.GenerateContext(scenario.Prompt)
	//fmt.Printf("%v", ScenarioFromFile(inputPath))
	/*
	tests := GenerateTestsFromFile(inputPath)
	fmt.Printf("== %v tests generated from %v ==\n", len(tests), inputPath)
	for testIdx := range tests {
		fmt.Printf("== Performing test %v / %v ==\n", testIdx+1, len(tests))
		tests[testIdx].Perform()
	}*/
}