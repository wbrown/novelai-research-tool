package nrt

import (
	"encoding/json"
	"fmt"
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
	Model                  []string  `json:"model"`
	Prefix                 []string  `json:"prefix"`
	PromptFilename         []string  `json:"prompt_filename"`
	Temperature            []float64 `json:"temperature"`
	MaxLength              []uint    `json:"max_length"`
	MinLength              []uint    `json:"min_length"`
	TopK                   []uint    `json:"top_k"`
	TopP                   []float64 `json:"top_p"`
	TailFreeSampling       []float64 `json:"tail_free_sampling"`
	RepetitionPenalty      []float64 `json:"repetition_penalty"`
	RepetitionPenaltyRange []uint    `json:"repetition_penalty_range"`
	RepetitionPenaltySlope []float64 `json:"repetition_penalty_slope"`
}

type ContentTest struct {
	OutputPrefix   string                        `json:"output_prefix"`
	PromptFilename string                        `json:"prompt_filename"`
	Iterations     int                           `json:"iterations"`
	Generations    int                           `json:"generations"`
	Parameters     novelai_api.NaiGenerateParams `json:"parameters"`
	Permutations   []PermutationsSpec            `json:"permutations"`
	Prompt         string
	WorkingDir     string
	PromptPath     string
	API            novelai_api.NovelAiAPI
}

type EncodedIterationResult struct {
	Prompt    string   `json:"prompt""`
	Responses []string `json:"responses""`
}

type IterationResult struct {
	Parameters novelai_api.NaiGenerateParams `json:"settings"`
	Prompt     string                        `json:"prompt"`
	Result     string                        `json:"result"`
	Responses  []string                      `json:"responses"`
	Encoded    EncodedIterationResult        `json:"encoded"`
}

func (ct *ContentTest) performGenerations(generations int, input string,
	report *ConsoleReporter) (results IterationResult) {
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
		report.ReportGeneration(resp.Response)
		context = context + resp.Response
		<-throttle.C
		throttle = time.NewTimer(1100 * time.Millisecond)
	}
	results.Result = strings.Join(results.Responses, "")
	return results
}


func (ct ContentTest) GeneratePermutationsFromSpec(spec PermutationsSpec) (tests []ContentTest) {
	permutations := []novelai_api.NaiGenerateParams{ct.Parameters}
	// Loop over the fields in `Permutations` type.
	fields := reflect.TypeOf(spec)
	for field := 0; field < fields.NumField(); field++ {
		// For each field, check the field contents and determine if there's any
		// values in there.
		fieldName := fields.FieldByIndex([]int{field}).Name
		fieldValues := reflect.ValueOf(spec).Field(field)
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
					// The only model that has `prefix`es is `6B-v3`, and we
					// don't want to do unnecessary work, so:
					//   If we are trying to create a permutation that has a
					//   `prefix` that is *not* `vanilla`, and we're permuting
					//   for a `model` with a value *other* than `6B-v3`, drop.
					if fieldName == "Prefix" && value.String() != "vanilla" &&
						permutation.Model != "6B-v3" {
						continue
					}
					newPermutations = append(newPermutations, permutation)
				}
			}
			permutations = newPermutations
		}
	}
	for permutationIdx := range permutations {
		newTest := ct
		newTest.Parameters = permutations[permutationIdx]
		// Pull up any `PromptFilename` values from the Parameters,
		// to the test and do setup.
		if len(newTest.Parameters.PromptFilename) > 0 {
			newTest.PromptFilename = newTest.Parameters.PromptFilename
			newTest.PromptPath = filepath.Join(newTest.WorkingDir, newTest.PromptFilename)
			if _, err := os.Stat(ct.PromptPath); os.IsNotExist(err) {
				fmt.Printf("`%v` does not exist!\n", ct.PromptPath)
				os.Exit(1)
			}
			newTest.loadPrompt(newTest.PromptPath)
		}
		tests = append(tests, newTest)
	}
	return tests
}

func (ct ContentTest) GeneratePermutations() (tests []ContentTest) {
	for specIdx := 0; specIdx < len(ct.Permutations); specIdx++ {
		tests = append(tests,
			ct.GeneratePermutationsFromSpec(ct.Permutations[specIdx])...)
	}
	return tests
}

func (ct *ContentTest) generateOutputPath() string {
	return filepath.Join(ct.WorkingDir,
		strings.Join([]string{
			ct.OutputPrefix,
			strings.Replace(
				strings.Replace(ct.Parameters.Model, "-", "_", -1),
				".", "_", -1),
			ct.Parameters.Prefix,
			time.Now().Format("2006-01-02T150405Z0700")}, "-"))
}

func (ct *ContentTest) loadPrompt(path string) {
	promptBytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	ct.Prompt = string(promptBytes)
}

func (ct ContentTest) Perform() {
	ct.loadPrompt(ct.PromptPath)
	consoleReport := ct.CreateConsoleReporter()
	textReport := ct.CreateTextReporter(ct.generateOutputPath() + ".txt")
	defer textReport.close()
	jsonReport := CreateJSONReporter(ct.generateOutputPath() + ".json")
	defer jsonReport.close()
	for iteration := 0; iteration < ct.Iterations; iteration++ {
		consoleReport.ReportIteration(iteration)
		responses := ct.performGenerations(ct.Generations, ct.Prompt, &consoleReport)
		textReport.write(responses.Result)
		jsonReport.write(&responses)
	}
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
	test.loadPrompt(test.PromptPath)
	return test.GeneratePermutations()
}

