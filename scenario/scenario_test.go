package scenario

import (
	"encoding/json"
	gpt_bpe "github.com/wbrown/novelai-research-tool/gpt-bpe"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"
)

var encoder gpt_bpe.GPTEncoder

const scenarioPath = "../tests/a_laboratory_assistant.scenario"

func AssertEqual(t *testing.T, a interface{}, b interface{}) {
	if reflect.DeepEqual(a, b) {
		return
	}
	t.Errorf("Received %v (type %v), expected %v (type %v)",
		a, reflect.TypeOf(a), b, reflect.TypeOf(b))
}

type jsonMap map[string]interface{}

func readJson(path string) (m jsonMap) {
	jsonBytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("scenario_test: Error loading JSON specification file `%s`: %v", path, err)
		os.Exit(1)
	}
	err = json.Unmarshal(jsonBytes, &m)
	if err != nil {
		log.Printf("scenario_test: Error loading JSON specification file `%s`: %v", path, err)
		os.Exit(1)
	}
	return m
}

func TestScenarioFromFile(t *testing.T) {
	if _, err := ScenarioFromFile(&encoder, "nonexistent.scenario"); err == nil {
		t.Errorf("Failed to handle nonexistent scenario files.")
	}
	var sc Scenario
	var err error
	if sc, err = ScenarioFromFile(&encoder, scenarioPath); err != nil {
		t.Errorf("Failed to load scenario file: %v", err)
	}
	jm := readJson(scenarioPath)
	AssertEqual(t, sc.Title, jm["title"].(string))
	AssertEqual(t, sc.Prompt, jm["prompt"].(string))
	AssertEqual(t, sc.Description, jm["description"].(string))
	if len(sc.Context) != 2 {
		t.Errorf("There should always be two contexts! %v", sc.Context)
	}
	AssertEqual(t, sc.Context[0].Label, "Memory")
	AssertEqual(t, sc.Context[1].Label, "A/N")
}

func TestScenario_ResolveLorebook(t *testing.T) {
	var sc Scenario
	var err error
	if sc, err = ScenarioFromFile(&encoder, scenarioPath); err != nil {
		t.Errorf("Failed to load scenario file: %v", err)
	}
	storyContext := sc.createStoryContext(sc.Prompt)
	lbEntries := sc.ResolveLorebook(ContextEntries{storyContext})
	for lbIdx := range lbEntries {
		lbEntry := lbEntries[lbIdx]
		for matchIdx := range lbEntry.MatchIndexes {
			for key, idxes := range lbEntry.MatchIndexes[matchIdx] {
				idx := idxes[0]
				promptSlice := sc.Prompt[idx[0]:idx[1]]
				if !strings.Contains(promptSlice, key) {
					t.Errorf("story[%v:%v] == '%v' != %v",
						idx[0], idx[1], promptSlice, key)
				}
			}
		}
	}
}

func TestScenario_GenerateContext(t *testing.T) {
	var sc Scenario
	var err error
	if sc, err = ScenarioFromFile(&encoder, scenarioPath); err != nil {
		t.Errorf("Failed to load scenario file: %v", err)
	}
	sc.GenerateContext(sc.Prompt, 1024)
}

func TestMain(m *testing.M) {
	encoder = gpt_bpe.NewEncoder()
	m.Run()
}