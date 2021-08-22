package nrt

import (
	"encoding/json"
	"fmt"
	"github.com/wbrown/novelai-research-tool/aimodules"
	novelai_api "github.com/wbrown/novelai-research-tool/novelai-api"
	"github.com/wbrown/novelai-research-tool/scenario"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"
)

type PlaceholderMap map[string]string

func (ph *PlaceholderMap) toMap() (ret map[string]string) {
	ret = make(map[string]string, 0)
	for k, v := range *ph {
		ret[k] = v
	}
	return ret
}

type PermutationsSpec struct {
	Model                  []string         `json:"model"`
	Prefix                 []string         `json:"prefix"`
	ModuleFilename         []string         `json:"module_filename"`
	PromptFilename         []string         `json:"prompt_filename"`
	Prompt                 []string         `json:"prompt"`
	Memory                 []string         `json:"memory"`
	AuthorsNote            []string         `json:"authors_note"`
	Placeholders           []PlaceholderMap `json:"placeholders"`
	Temperature            []*float64       `json:"temperature"`
	MaxLength              []*uint          `json:"max_length"`
	MinLength              []*uint          `json:"min_length"`
	TopK                   []*uint          `json:"top_k"`
	TopP                   []*float64       `json:"top_p"`
	TailFreeSampling       []*float64       `json:"tail_free_sampling"`
	RepetitionPenalty      []*float64       `json:"repetition_penalty"`
	RepetitionPenaltyRange []*uint          `json:"repetition_penalty_range"`
	RepetitionPenaltySlope []*float64       `json:"repetition_penalty_slope"`
}

type ContentTest struct {
	OutputPrefix     string                        `json:"output_prefix"`
	PromptFilename   string                        `json:"prompt_filename"`
	ScenarioFilename string                        `json:"scenario_filename"`
	ModuleFilename   string                        `json:"module_filename"`
	Prompt           string                        `json:"prompt"`
	Memory           string                        `json:"memory"`
	AuthorsNote      string                        `json:"authors_note"`
	MaxTokens        *int                          `json:"max_tokens"`
	Iterations       *int                          `json:"iterations"`
	Generations      *int                          `json:"generations"`
	Parameters       novelai_api.NaiGenerateParams `json:"parameters"`
	Permutations     []PermutationsSpec            `json:"permutations"`
	Placeholders     PlaceholderMap                `json:"placeholders"`
	WorkingDir       string
	PromptPath       string
	ScenarioPath     string
	ModulePath       string
	Scenario         *scenario.Scenario
	AIModule         *aimodules.AIModule
	API              novelai_api.NovelAiAPI
}

func MakeDefaultContentTest() (ct ContentTest) {
	maxTokens := 2048
	iterations := 10
	generations := 25
	ct.OutputPrefix = "./"
	ct.MaxTokens = &maxTokens
	ct.Iterations = &iterations
	ct.Generations = &generations
	ct.WorkingDir = "./"
	return ct
}

func (ct *ContentTest) CoerceContentTest(other *ContentTest) {
	if ct.OutputPrefix == "" {
		ct.OutputPrefix = other.OutputPrefix
	}
	if ct.PromptFilename == "" {
		ct.PromptFilename = other.PromptFilename
	}
	if ct.OutputPrefix == "" {
		ct.OutputPrefix = other.OutputPrefix
	}
	if ct.MaxTokens == nil {
		ct.MaxTokens = other.MaxTokens
	}
	if ct.Iterations == nil {
		ct.Iterations = other.Iterations
	}
	if ct.Generations == nil {
		ct.Generations = other.Generations
	}
}

type ContentTests []ContentTest

type RequestContext struct {
	Request       novelai_api.NaiGenerateResp `json:"requests"`
	ContextReport scenario.ContextReport      `json:"context_report"`
}

type EncodedIterationResult struct {
	Prompt      string           `json:"prompt"`
	Memory      string           `json:"memory"`
	AuthorsNote string           `json:"authors_note"`
	Requests    []RequestContext `json:"requests"`
}

type IterationResult struct {
	Parameters    novelai_api.NaiGenerateParams `json:"settings"`
	Prompt        string                        `json:"prompt"`
	Memory        string                        `json:"memory"`
	AuthorsNote   string                        `json:"authors_note"`
	Result        string                        `json:"result"`
	Responses     []string                      `json:"responses"`
	ContextReport scenario.ContextReport        `json:"context_report"`
	Encoded       EncodedIterationResult        `json:"encoded"`
}

func (ct *ContentTest) performGenerations(generations int, input string,
	reporters *Reporters) (results IterationResult) {
	context := input
	results.Prompt = input
	results.Memory = ct.Memory
	results.AuthorsNote = ct.AuthorsNote
	results.Parameters = ct.Parameters
	ct.Scenario.SetMemory(ct.Memory)
	ct.Scenario.SetAuthorsNote(ct.AuthorsNote)
	throttle := time.NewTimer(1100 * time.Millisecond)
	for generation := 0; generation < generations; generation++ {
		submission, ctxReport := ct.Scenario.GenerateContext(context, *ct.MaxTokens)
		if generation == generations-1 {
			trimTrue := true
			ct.Parameters.TrimResponses = &trimTrue
		}
		resp := ct.API.GenerateWithParams(&submission, ct.Parameters)
		if generation == 0 {
			results.Encoded.Prompt = resp.EncodedRequest
		}
		results.Responses = append(results.Responses, resp.Response)
		results.Encoded.Requests = append(results.Encoded.Requests,
			RequestContext{resp, ctxReport})
		reporters.ReportGeneration(resp.Response)
		context = context + resp.Response
		<-throttle.C
		throttle = time.NewTimer(1100 * time.Millisecond)
	}
	results.Result = strings.Join(results.Responses, "")
	return results
}

func makeFileNameSafe(s string) string {
	return strings.NewReplacer(
		"-", "_",
		".", "_",
		":", "_",
		" ", "_").Replace(s)
}

func (ct *ContentTest) GetModuleFilename() (moduleFile string) {
	if ct.ModuleFilename != "" {
		return ct.ModuleFilename
	} else if ct.Scenario != nil && ct.Scenario.AIModule != nil &&
		ct.Parameters.Prefix == ct.Scenario.AIModule.ToPrefix() &&
		ct.ScenarioFilename != "" {
		return ct.ScenarioFilename
	} else {
		return "unknown"
	}
}

func (ct *ContentTest) GetPrefixName() (prefixName string) {
	if ct.Scenario != nil && ct.Scenario.AIModule != nil &&
		ct.Parameters.Prefix == ct.Scenario.AIModule.ToPrefix() &&
		len(ct.Scenario.AIModule.Name) != 0 {
		prefixName = ct.Scenario.AIModule.Name
	} else if ct.AIModule != nil {
		prefixName = ct.AIModule.Name
	} else {
		prefixName = ct.Parameters.Prefix
	}
	return prefixName
}

func (ct *ContentTest) MakeLabel(spec PermutationsSpec) (label string) {
	fieldNames := make([]string, 0)
	fields := reflect.TypeOf(spec)
	for field := 0; field < fields.NumField(); field++ {
		fieldName := fields.FieldByIndex([]int{field}).Name
		fieldValues := reflect.ValueOf(spec).Field(field)
		if fieldValues.Len() > 0 {
			fieldNames = append(fieldNames, fieldName)
		}
	}
	ctFields := reflect.TypeOf(ct.Parameters)
	for fieldIdx := range fieldNames {
		if len(label) != 0 {
			label += ","
		}
		fieldName := fieldNames[fieldIdx]
		fieldValueRepr := "#0"
		switch fieldName {
		case "Prefix":
			fieldValueRepr = ct.GetPrefixName()
		case "Placeholders":
			for placeholderIdx := range spec.Placeholders {
				if reflect.DeepEqual(spec.Placeholders[placeholderIdx], ct.Placeholders) {
					fieldValueRepr = fmt.Sprintf("#%d", placeholderIdx+1)
					break
				}
			}
		case "Memory":
			for memoryIdx := range spec.Memory {
				if spec.Memory[memoryIdx] == ct.Memory {
					fieldValueRepr = fmt.Sprintf("#%d", memoryIdx+1)
					break
				}
			}
		case "AuthorsNote":
			for authIdx := range spec.AuthorsNote {
				if spec.AuthorsNote[authIdx] == ct.AuthorsNote {
					fieldValueRepr = fmt.Sprintf("#%d", authIdx+1)
					break
				}
			}
		case "Prompt":
			for promptIdx := range spec.Prompt {
				if spec.Prompt[promptIdx] == ct.Prompt {
					fieldValueRepr = fmt.Sprintf("#%d", promptIdx+1)
					break
				}
			}
		case "ModuleFilename":
			fieldValueRepr = ct.GetModuleFilename()
		case "PromptFilename":
			fieldValueRepr = strings.Replace(
				strings.Replace(
					filepath.Base(fmt.Sprintf("%v",
						ct.PromptFilename)), "-", "_", -1),
				".", "_", -1)
		default:
			field, _ := ctFields.FieldByName(fieldName)
			ctVal := reflect.ValueOf(ct.Parameters).Field(field.Index[0])
			if ctVal.Kind() == reflect.Ptr {
				fieldValueRepr = fmt.Sprintf("%v", ctVal.Elem())
			} else {
				fieldValueRepr = fmt.Sprintf("%v", ctVal)
			}
		}
		fieldValueRepr = makeFileNameSafe(fmt.Sprintf("%v", fieldValueRepr))
		label += fieldName + "=" + fieldValueRepr
	}
	return label
}

func (ct ContentTest) FieldsSame(fields []string, other ContentTest) bool {
	ctFields := reflect.TypeOf(ct.Parameters)
	for fieldIdx := range fields {
		fieldName := fields[fieldIdx]
		switch fieldName {
		case "Placeholders":
			if !reflect.DeepEqual(ct.Placeholders, other.Placeholders) {
				return false
			}
			continue
		case "Memory":
			if ct.Memory != other.Memory {
				return false
			}
			continue
		case "AuthorsNote":
			if ct.AuthorsNote != other.AuthorsNote {
				return false
			}
			continue
		case "ModuleFilename":
			if ct.ModuleFilename != other.ModuleFilename {
				return false
			}
			continue
		case "PromptFilename":
			if ct.PromptFilename != other.PromptFilename {
				return false
			}
			continue
		case "Prompt":
			if ct.Prompt != other.Prompt {
				return false
			}
			continue
		}
		field, _ := ctFields.FieldByName(fieldName)
		ctVal := reflect.ValueOf(ct.Parameters).Field(field.Index[0])
		otherVal := reflect.ValueOf(other.Parameters).Field(field.Index[0])
		if ctVal.Kind() == reflect.Ptr {
			if fmt.Sprintf("%v", ctVal.Elem()) !=
				fmt.Sprintf("%v", otherVal.Elem()) {
				return false
			}
		} else if fmt.Sprintf("%v", ctVal) != fmt.Sprintf("%v", otherVal) {
			return false
		}
	}
	return true
}

func resolvePermutation(origPermutation ContentTest,
	fieldName string, fieldValues *reflect.Value) ContentTests {
	newPermutations := make(ContentTests, 0)
	for valIdx := 0; valIdx < fieldValues.Len(); valIdx++ {
		permutation := origPermutation
		value := fieldValues.Index(valIdx)
		targetField, _ := reflect.TypeOf(permutation.Parameters).FieldByName(fieldName)
		switch fieldName {
		case "Placeholders":
			newPlaceholders := make(PlaceholderMap, 0)
			fromPlaceholders := value.Interface().(PlaceholderMap)
			for k, v := range permutation.Placeholders {
				newPlaceholders[k] = v
			}
			for k, v := range fromPlaceholders {
				newPlaceholders[k] = v
			}
			permutation.Placeholders = newPlaceholders
		case "Prompt":
			permutation.Prompt = fmt.Sprintf("%s", value)
		case "Memory":
			permutation.Memory = fmt.Sprintf("%s", value)
		case "AuthorsNote":
			permutation.AuthorsNote = fmt.Sprintf("%s", value)
		case "PromptFilename":
			permutation.PromptFilename = fmt.Sprintf("%v", value)
			if len(permutation.PromptFilename) > 0 {
				permutation.PromptPath = filepath.Join(permutation.WorkingDir,
					permutation.PromptFilename)
				if _, err := os.Stat(permutation.PromptPath); os.IsNotExist(err) {
					log.Printf("nrt: Prompt file `%s` does not exist!\n",
						permutation.PromptPath)
					os.Exit(1)
				}
				permutation.loadPrompt(permutation.PromptPath)
			}
		case "ModuleFilename":
			permutation.ModuleFilename = fmt.Sprintf("%v", value)
			if len(permutation.ModuleFilename) > 0 {
				permutation.ModulePath = filepath.Join(permutation.WorkingDir,
					permutation.ModuleFilename)
				if _, err := os.Stat(permutation.ModulePath); os.IsNotExist(err) {
					log.Printf("nrt: Module file `%s` does not exist!\n",
						permutation.ModulePath)
					os.Exit(1)
				}
				aiModule := aimodules.AIModuleFromFile(permutation.ModulePath)
				permutation.AIModule = &aiModule
				permutation.Scenario.Settings.Prefix = permutation.AIModule.ToPrefix()
				permutation.Scenario.Settings.ScenarioAIModule = &scenario.ScenarioAIModule{
					Name: aiModule.Name,
					Id: permutation.Scenario.Settings.Prefix,
					Description: aiModule.Description,
				}
			}
		case "Model":
			permutation.Parameters.Model = value.String()
		default:
			reflect.ValueOf(&permutation.Parameters).Elem().Field(targetField.Index[0]).Set(value)
		}
		newPermutations = append(newPermutations, permutation)
	}
	return newPermutations
}

func (ct ContentTest) GeneratePermutationsFromSpec(spec PermutationsSpec) ContentTests {
	templateTest := ct
	templateTest.Parameters = ct.Parameters
	permutations := ContentTests{templateTest}
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
			newPermutations := make(ContentTests, 0)
			// Loop over each permutation we already have existing.
			for permutationTargetIdx := range permutations {
				// Create a new permutation for each value in the field.
				permutation := permutations[permutationTargetIdx]
				permScen := *permutation.Scenario
				permScen.PlaceholderMap = permScen.PlaceholderMap
				permutation.Scenario = &permScen
				newPermutations = append(newPermutations,
					resolvePermutation(permutation, fieldName, &fieldValues)...)
			}
			permutations = newPermutations
		}
	}
	ct.Parameters.Label = ct.MakeLabel(spec)
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
			permutation.Scenario.Settings.Parameters = permutation.Parameters
			permutation.Parameters.Label = permutation.MakeLabel(spec)
			filteredPermutations = append(filteredPermutations, permutation)
		}
	}
	return filteredPermutations
}

func (ct ContentTest) GeneratePermutations() (tests []ContentTest) {
	if len(ct.Permutations) > 0 {
		for specIdx := 0; specIdx < len(ct.Permutations); specIdx++ {
			tests = append(tests,
				ct.GeneratePermutationsFromSpec(ct.Permutations[specIdx])...)
		}
	} else {
		if ct.Parameters.Label == "" {
			if ct.ScenarioFilename != "" {
				ct.Parameters.Label = strings.Replace(
					strings.Replace(
						filepath.Base(fmt.Sprintf("%v",
							ct.ScenarioFilename)), "-", "_", -1),
					".", "_", -1)
			} else {
				ct.Parameters.Label = "base"
			}
		}
		tests = append(tests, ct)
	}
	return tests
}

const MaxFilePathLength = 210
const MaxFileExtensionLength = 5

func (ct *ContentTest) generateOutputPath() string {
	tsString := ",TS=" + time.Now().Format("2006-01-02T1504")
	budget := MaxFilePathLength -
		(len(filepath.Join(ct.WorkingDir, ct.OutputPrefix)) +
			len(tsString) + len(ct.WorkingDir) +
			MaxFileExtensionLength + 1)
	label := ct.Parameters.Label
	if budget < len(label) && runtime.GOOS == "windows" {
		if budget < 30 {
			log.Printf("nrt: your working path is too long: %v",
				ct.WorkingDir)
			os.Exit(1)
		}
		label = label[:budget]
	}
	return filepath.Join(ct.WorkingDir,
		ct.OutputPrefix+"-"+label+tsString)
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
	// ct.loadPrompt(ct.PromptPath)
	ct.Scenario.PlaceholderMap.UpdateValues(ct.Placeholders.toMap())
	ct.Prompt = ct.Scenario.PlaceholderMap.ReplacePlaceholders(ct.Prompt)
	ct.Memory = ct.Scenario.PlaceholderMap.ReplacePlaceholders(ct.Memory)
	ct.AuthorsNote = ct.Scenario.PlaceholderMap.ReplacePlaceholders(ct.AuthorsNote)
	reporters := ct.MakeReporters()
	defer reporters.close()
	for iteration := 0; iteration < *ct.Iterations; iteration++ {
		reporters.ReportIteration(iteration)
		responses := ct.performGenerations(*ct.Generations, ct.Prompt, &reporters)
		reporters.SerializeIteration(&responses)
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
	if test.OutputPrefix == "" {
		log.Println("nrt: `output_prefix` must be set to a non-empty string.\n")
		os.Exit(1)
	} else if test.PromptFilename == "" && test.Prompt == "" && test.Memory == "" &&
		test.AuthorsNote == "" && test.ScenarioFilename == "" {
		log.Printf(
			"nrt: %s %s\n",
			"at least one of prompt_filename, prompt, memory, authors_note, or",
			"scenario_filename must be filled in.")
		os.Exit(1)
	} else if test.PromptFilename != "" && test.Prompt != "" {
		log.Println("nrt: you cannot have both `prompt_filename` and `prompt` set")
		os.Exit(1)
	}
	test.WorkingDir = filepath.Dir(path)
	if test.ScenarioFilename != "" {
		test.ScenarioPath = filepath.Join(test.WorkingDir, test.ScenarioFilename)
		if _, err = os.Stat(test.ScenarioPath); os.IsNotExist(err) {
			log.Printf("nrt: Scenario file `%v` does not exist!\n", test.ScenarioPath)
			os.Exit(1)
		}
		fmt.Printf("ScenarioPath: %v\n", test.ScenarioPath)
		if scenario, err := scenario.ScenarioFromFile(nil, test.ScenarioPath); err != nil {
			log.Printf("nrt: Error loading scenario: %v\n", err)
			os.Exit(1)
		} else {
			test.Scenario = &scenario
		}
		test.Prompt = test.Scenario.Prompt
		test.Memory = test.Scenario.Context[0].Text
		test.AuthorsNote = test.Scenario.Context[1].Text
		test.Parameters.CoerceNullValues(test.Scenario.Settings.Parameters)
	} else {
		test.Parameters.CoerceDefaults()
	}
	if test.ModuleFilename != "" {
		test.ModulePath = filepath.Join(test.WorkingDir,
			test.ModuleFilename)
		if _, err := os.Stat(test.ModulePath); os.IsNotExist(err) {
			log.Printf("nrt: Module file `%s` does not exist!\n",
				test.ModulePath)
			os.Exit(1)
		}
		aimodule := aimodules.AIModuleFromFile(test.ModulePath)
		test.AIModule = &aimodule
		test.Parameters.Prefix = test.AIModule.ToPrefix()
	}
	if test.PromptFilename != "" {
		test.PromptPath = filepath.Join(test.WorkingDir, test.PromptFilename)
		if _, err := os.Stat(test.PromptPath); os.IsNotExist(err) {
			log.Printf("nrt: Prompt file `%v` does not exist!\n", test.PromptPath)
			os.Exit(1)
		}
		test.loadPrompt(test.PromptPath)
	}
	if test.ScenarioFilename == "" {
		scenarioSpec := scenario.ScenarioFromSpec(test.Prompt, test.Memory,
			test.AuthorsNote)
		test.Scenario = &scenarioSpec
		test.Scenario.Settings.Parameters.CoerceNullValues(test.Parameters)
	}
	defaultTest := MakeDefaultContentTest()
	test.CoerceContentTest(&defaultTest)
	return test
}

func MakeTestFromScenario(path string) (test ContentTest) {
	test = MakeDefaultContentTest()
	test.ScenarioPath = path
	if _, err := os.Stat(test.ScenarioPath); os.IsNotExist(err) {
		log.Printf("nrt: Scenario file `%v` does not exist!\n", test.ScenarioPath)
		os.Exit(1)
	}
	fmt.Printf("ScenarioPath: %v\n", test.ScenarioPath)
	if sc, err := scenario.ScenarioFromFile(nil, test.ScenarioPath); err != nil {
		log.Printf("nrt: Error loading scenario: %v\n", err)
		os.Exit(1)
	} else {
		test.Scenario = &sc
	}
	test.WorkingDir = filepath.Dir(path)
	test.OutputPrefix = strings.Replace(filepath.Base(path), ".scenario", "", -1)
	test.Prompt = test.Scenario.Prompt
	test.Memory = test.Scenario.Context[0].Text
	test.AuthorsNote = test.Scenario.Context[1].Text
	test.Scenario.Settings.Parameters.CoerceDefaults()
	test.Parameters.CoerceNullValues(test.Scenario.Settings.Parameters)
	test.Parameters.Prefix = test.Scenario.Settings.Prefix
	*test.Parameters.BanBrackets = test.Scenario.Settings.BanBrackets
	test.Parameters = test.Scenario.Settings.Parameters
	return test
}

func GenerateTestsFromFile(path string) (tests []ContentTest) {
	if strings.HasSuffix(path, ".scenario") {
		test := MakeTestFromScenario(path)
		test.API = novelai_api.NewNovelAiAPI()
		tests = []ContentTest{test}
	} else {
		test := LoadSpecFromFile(path)
		test.API = novelai_api.NewNovelAiAPI()
		tests = test.GeneratePermutations()
	}
	return tests
}
