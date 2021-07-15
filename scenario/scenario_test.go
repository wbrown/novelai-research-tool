package scenario

import (
	"encoding/json"
	gpt_bpe "github.com/wbrown/novelai-research-tool/gpt-bpe"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"testing"
)

var encoder gpt_bpe.GPTEncoder

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

func TestScenario_ResolveLorebook(t *testing.T) {

}

func TestScenarioFromFile(t *testing.T) {
	if _, err := ScenarioFromFile(&encoder, "nonexistent.scenario"); err == nil {
		t.Errorf("Failed to handle nonexistent scenario files.")
	}
	var sc Scenario
	var err error
	if sc, err = ScenarioFromFile(&encoder,
		"../tests/a_laboratory_assistant.scenario"); err != nil {
		t.Errorf("Failed to load scenario file: %v", err)
	}
	jm := readJson("../tests/a_laboratory_assistant.scenario")
	AssertEqual(t, sc.Title, jm["Title"].(string))
	AssertEqual(t, sc.Prompt, jm["Prompt"])
	if len(sc.Context) != 2 {
		t.Errorf("There should always be two contexts! %v", sc.Context)
	}

}

func TestMain(m *testing.M) {
	encoder = gpt_bpe.NewEncoder()
	m.Run()
}