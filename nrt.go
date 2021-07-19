package nrt

import (
	"encoding/json"
	"fmt"
	novelai_api "github.com/wbrown/novelai-research-tool/novelai-api"
	"github.com/wbrown/novelai-research-tool/scenario"
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
	Memory                 []string  `json:"memory"`
	AuthorsNote            []string  `json:"authors_note"`
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
	Memory         string                        `json:"memory"`
	AuthorsNote    string                        `json:"authors_note"`
	MaxTokens      int                           `json:"max_tokens"`
	Iterations     int                           `json:"iterations"`
	Generations    int                           `json:"generations"`
	Parameters     novelai_api.NaiGenerateParams `json:"parameters"`
	Permutations   []PermutationsSpec            `json:"permutations"`
	Prompt         string
	WorkingDir     string
	PromptPath     string
	Scenario       scenario.Scenario
	API            novelai_api.NovelAiAPI
}

type EncodedIterationResult struct {
	Prompt      string                        `json:"prompt""`
	Memory      string                        `json:"memory"`
	AuthorsNote string                        `json:"authors_note"`
	Requests    []novelai_api.NaiGenerateResp `json:"requests"`
}

type IterationResult struct {
	Parameters  novelai_api.NaiGenerateParams `json:"settings"`
	Prompt      string                        `json:"prompt"`
	Memory      string                        `json:"memory"`
	AuthorsNote string                        `json:"authors_note"`
	Result      string                        `json:"result"`
	Responses   []string                      `json:"responses"`
	Activations []scenario.ContextReport      `json:"context_report"`
	Encoded     EncodedIterationResult        `json:"encoded"`
}

func (ct *ContentTest) performGenerations(generations int, input string,
	report *ConsoleReporter) (results IterationResult) {
	context := input
	results.Prompt = input
	results.Memory = ct.Memory
	results.AuthorsNote = ct.AuthorsNote
	results.Parameters = ct.Parameters
	throttle := time.NewTimer(1100 * time.Millisecond)
	for generation := 0; generation < generations; generation++ {
		submission := ct.Scenario.GenerateContext(context, ct.MaxTokens)
		resp := ct.API.GenerateWithParams(&submission, ct.Parameters)
		if generation == 0 {
			results.Encoded.Prompt = resp.EncodedRequest
		}
		results.Responses = append(results.Responses, resp.Response)
		results.Encoded.Requests = append(results.Encoded.Requests, resp)
		report.ReportGeneration(resp.Response)
		context = context + resp.Response
		<-throttle.C
		throttle = time.NewTimer(1100 * time.Millisecond)
	}
	results.Result = strings.Join(results.Responses, "")
	return results
}

func (ct ContentTest) FieldsSame(fields []string, other ContentTest) bool {
	ctFields := reflect.TypeOf(ct.Parameters)
	for fieldIdx := range fields {
		fieldName := fields[fieldIdx]
		switch fieldName {
		case "Memory":
			if ct.Memory != other.Memory {
				return false
			}
		case "AuthorsNote":
			if ct.AuthorsNote != other.AuthorsNote {
				return false
			}
		case "PromptFilename":
			if ct.PromptFilename != other.PromptFilename {
				return false
			}
		}
		field, _ := ctFields.FieldByName(fieldName)
		ctVal := reflect.ValueOf(ct.Parameters).Field(field.Index[0])
		otherVal := reflect.ValueOf(other.Parameters).Field(field.Index[0])
		if fmt.Sprintf("%v", ctVal) != fmt.Sprintf("%v", otherVal) {
			return false
		}
	}
	return true
}

func (ct ContentTest) GeneratePermutationsFromSpec(spec PermutationsSpec) []ContentTest {
	templateTest := ct
	templateTest.Parameters = ct.Parameters
	permutations := []ContentTest{templateTest}
	// Loop over the fields in `Permutations` type.
	fields := reflect.TypeOf(spec)
	fieldNames := make([]string, 0)
	for field := 0; field < fields.NumField(); field++ {
		// For each field, check the field contents and determine if there's any
		// values in there.
		fieldName := fields.FieldByIndex([]int{field}).Name
		fieldValues := reflect.ValueOf(spec).Field(field)
		if fieldValues.Len() > 0 {
			fieldNames = append(fieldNames, fieldName)
			// Loop over the values in the field to permute on.
			newPermutations := make([]ContentTest, 0)
			// Loop over each permutation we already have existing.
			for permutationTargetIdx := range permutations {
				// Create a new permutation for each value in the field.
				for valIdx := 0; valIdx < fieldValues.Len(); valIdx++ {
					value := fieldValues.Index(valIdx)
					permutation := permutations[permutationTargetIdx]
					targetField, _ := reflect.TypeOf(permutation.Parameters).FieldByName(fieldName)
					var fieldValueRepr string
					switch fieldName {
					case "Memory":
						permutation.Memory = fmt.Sprintf("%s", value)
						fieldValueRepr = fmt.Sprintf("#%d", valIdx)
					case "AuthorsNote":
						permutation.AuthorsNote = fmt.Sprintf("%s", value)
						fieldValueRepr = fmt.Sprintf("#%d", valIdx)
					case "PromptFilename":
						permutation.PromptFilename = fmt.Sprintf("%v", value)
						permutation.PromptPath = filepath.Join(permutation.WorkingDir,
							permutation.PromptFilename)
						if _, err := os.Stat(permutation.PromptPath); os.IsNotExist(err) {
							log.Printf("nrt: Prompt file `%s` does not exist!\n", ct.PromptPath)
							os.Exit(1)
						}
						fieldValueRepr = fmt.Sprintf("%s", value)
						permutation.loadPrompt(permutation.PromptPath)
					default:
						reflect.ValueOf(&permutation.Parameters).Elem().Field(targetField.Index[0]).Set(
							value)
						fieldValueRepr = strings.Replace(
							strings.Replace(
								filepath.Base(fmt.Sprintf("%v",
									value)), "-", "_", -1),
							".", "_", -1)
					}
					// Update our label to reflect the field value we just permuted on.
					if len(permutation.Parameters.Label) != 0 {
						permutation.Parameters.Label += ","
					}
					permutation.Parameters.Label += fieldName + "=" + fieldValueRepr
					if fieldName == "Prefix" && value.String() != "vanilla" &&
						permutation.Parameters.Model != "6B-v3" {
						continue
					}
					newPermutations = append(newPermutations, permutation)
				}
			}
			permutations = newPermutations
		}
	}
	filteredPermutations := []ContentTest{ct}
	for permutationIdx := range permutations {
		permutation := permutations[permutationIdx]
		if permutation.Parameters.Model != "6B-v3" &&
			permutation.Parameters.Prefix != "vanilla" {
			continue
		}
		// Deduplicate based on fields we've permuted on.
		same := false
		for filteredIdx := range filteredPermutations {
			filteredPermutation := filteredPermutations[filteredIdx]
			if permutation.FieldsSame(fieldNames, filteredPermutation) {
				same = true
				break
			}
		}
		if !same {
			filteredPermutations = append(filteredPermutations, permutation)
		}
	}
	return filteredPermutations
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
			ct.Parameters.Label + ",TS=" +
				time.Now().Format("2006-01-02T1504")},
			"-"))
}

func (ct *ContentTest) loadPrompt(path string) {
	promptBytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("nrt: Error loading prompt file `%s`: %v", path, err)
		os.Exit(1)
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

func LoadSpecFromFile(path string) (test ContentTest) {
	configBytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("nrt: Error loading JSON specification file `%s`: %v", path, err)
		os.Exit(1)
	}
	err = json.Unmarshal(configBytes, &test)
	if err != nil {
		log.Printf("nrt: Error loading JSON specification file `%s`: %v", path, err)
		os.Exit(1)
	}
	if test.MaxTokens == 0 {
		test.MaxTokens = 2048
	}
	test.WorkingDir = filepath.Dir(path)
	test.PromptPath = filepath.Join(test.WorkingDir, test.PromptFilename)
	if _, err := os.Stat(test.PromptPath); os.IsNotExist(err) {
		log.Printf("nrt: Prompt file `%v` does not exist!\n", test.PromptPath)
		os.Exit(1)
	}
	test.loadPrompt(test.PromptPath)
	test.Scenario = scenario.ScenarioFromSpec(test.Prompt, test.Memory,
		test.AuthorsNote)
	return test
}

func GenerateTestsFromFile(path string) (tests []ContentTest) {
	test := LoadSpecFromFile(path)
	test.API = novelai_api.NewNovelAiAPI()
	return test.GeneratePermutations()
}
