package scenario

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"
)

const scenarioPath = "../tests/a_laboratory_assistant.scenario"
const frankensteinPath = "../tests/frankenstein.scenario"

func AssertEqual(t *testing.T, a interface{}, b interface{}) {
	if reflect.DeepEqual(a, b) {
		return
	}
	t.Errorf("\nReceived: %v (type %v),\nExpected: %v (type %v)",
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
	if _, err := ScenarioFromFile("nonexistent.scenario"); err == nil {
		t.Errorf("Failed to handle nonexistent scenario files.")
	}
	var sc Scenario
	var err error
	if sc, err = ScenarioFromFile(scenarioPath); err != nil {
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
	if sc, err = ScenarioFromFile(scenarioPath); err != nil {
		t.Errorf("Failed to load scenario file: %v", err)
	}
	storyContext := sc.createStoryContext(sc.Prompt)
	lbEntries := sc.Lorebook.ResolveContexts(&sc.PlaceholderMap,
		&ContextEntries{storyContext})
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

func StringifyContextReport(t *testing.T, ctxReport ContextReport) string {
	if reprBytes, err := json.MarshalIndent(ctxReport, "", "  "); err != nil {
		t.Errorf("Failed to unmarshal ContextReport to string")
	} else {
		return string(reprBytes)
	}
	return ""
}

const phTable = `%{
1Name[Daniel Blackthorn]:Name
2HairColor[red]:Hair Color (red, blonde)
3Skin[pale]:Complexion (dark, light, olive, pale)
}
I’m ${1Name}, a male soldier who fought in the American Civil War, on the Confederate side. When the South lost the War of Northern aggression, I went westward, and participated in the Indian Wars. Eventually, I ended up in San Francisco, and went overseas to Japan during the Edo Period. I was hired to and am currently training the soldiers of the Japanese daimyōs in rifle warfare and skirmishing tactics. I’d been there long enough and had a knack for languages to become nearly fluent in Japanese. I trained in the samurai arts of the katana, and the code of Bushido. I am known as the White Samurai, and my ${2HairColor} and ${3Skin} skin is viewed with near superstitious awe.`
const phTableExpected = "I’m Daniel Blackthorn, a male soldier who fought in the American Civil War, on the Confederate side. When the South lost the War of Northern aggression, I went westward, and participated in the Indian Wars. Eventually, I ended up in San Francisco, and went overseas to Japan during the Edo Period. I was hired to and am currently training the soldiers of the Japanese daimyōs in rifle warfare and skirmishing tactics. I’d been there long enough and had a knack for languages to become nearly fluent in Japanese. I trained in the samurai arts of the katana, and the code of Bushido. I am known as the White Samurai, and my red and pale skin is viewed with near superstitious awe."

type PlaceholderTest struct {
	input          string
	expected       string
	expectedStruct Placeholders
}

type PlaceholderTests []PlaceholderTest

var placeholderTests = PlaceholderTests{
	{"This is a foobar test. ${1Name[Daniel]:Your name?} ${2HerName[Audrey]:Her name?}",
		"This is a foobar test. Daniel Audrey",
		Placeholders{
			"1Name": {"1Name",
				"Daniel",
				"Your name?",
				"",
				"Daniel"},
			"2HerName": {"2HerName",
				"Audrey",
				"Her name?",
				"",
				"Audrey"},
		}},
	{phTable,
		phTableExpected,
		Placeholders{
			"1Name": {"1Name",
				"Daniel Blackthorn",
				"Name",
				"",
				"Daniel Blackthorn"},
			"2HairColor": {"2HairColor",
				"red",
				"Hair Color (red, blonde)",
				"",
				"red"},
			"3Skin": {"3Skin",
				"pale",
				"Complexion (dark, light, olive, pale)",
				"",
				"pale"}}},
}

func TestScenario_DiscoverPlaceholders(t *testing.T) {
	for testIdx := range placeholderTests {
		test := placeholderTests[testIdx]
		defs := DiscoverPlaceholderDefs(test.input)
		defs.merge(DiscoverPlaceholderTable(test.input))
		AssertEqual(t, defs, test.expectedStruct)
	}
}

func TestScenario_RealizePlaceholderDefs(t *testing.T) {
	for testIdx := range placeholderTests {
		test := placeholderTests[testIdx]
		sc := ScenarioFromSpec(test.input, "", "", "euterpe-v2")
		sc.Settings.Parameters.CoerceDefaults()
		ctx, _ := sc.GenerateContext(sc.Prompt, 1024)
		AssertEqual(t, ctx, test.expected)
	}
}

type GenerateContextTest struct {
	scenarioPath string
	budget       int
	expected     int
}

type GenerateContextTests []GenerateContextTest

var generateContextTests = GenerateContextTests{
	{scenarioPath, 2048, 11},
	{scenarioPath, 1024, 8},
	{frankensteinPath, 2048, 3},
}

func TestScenario_GenerateContext(t *testing.T) {
	var sc Scenario
	var err error

	for testIdx := range generateContextTests {
		test := generateContextTests[testIdx]
		if sc, err = ScenarioFromFile(test.scenarioPath); err != nil {
			t.Errorf("Failed to load scenario file: %v", err)
		} else {
			_, report := sc.GenerateContext(sc.Prompt, test.budget)
			if len(report) != test.expected {
				t.Log(StringifyContextReport(t, report))
				t.Errorf("%s, budget %d: Expected %d insertions, got %d",
					test.scenarioPath, test.budget, test.expected, len(report))
			}
		}
	}
}

func TestMain(m *testing.M) {
	m.Run()
}
