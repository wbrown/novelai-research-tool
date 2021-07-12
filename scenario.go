package main

import (
	"encoding/json"
	"fmt"
	gpt_bpe "github.com/wbrown/novelai-research-tool/gpt-bpe"
	novelai_api "github.com/wbrown/novelai-research-tool/novelai-api"
	"io/ioutil"
	"log"
	"regexp"
	"sort"
)

type ContextConfig struct {
	Prefix            string `json:"prefix"`
	Suffix            string `json:"suffix"`
	TokenBudget       int    `json:"tokenBudget"`
	ReservedTokens    int    `json:"reservedTokens"`
	BudgetPriority    int    `json:"budgetPriority"`
	TrimDirection     string `json:"trimDirection"`
	InsertionType     string `json:"insertionType"`
	MaximumTrimType   string `json:"maximumTrimType"`
	InsertionPosition int    `json:"insertionPosition"`
}

type ContextEntry struct {
	Text       string        `json:"text"`
	ContextCfg ContextConfig `json:"contextConfig"`
	Tokens     []uint16
	Label      string
	Indexes    [][]int
}

type ContextEntries []ContextEntry

func (ctxes ContextEntries) Len() int {
	return len(ctxes)
}

func (ctxes ContextEntries) Swap(i, j int) {
	ctxes[i], ctxes[j] = ctxes[j], ctxes[i]
}

func (ctxes ContextEntries) Less(i, j int) bool {
	return ctxes[i].ContextCfg.BudgetPriority <
		ctxes[j].ContextCfg.BudgetPriority
}


type LorebookEntry struct {
	Text                string        `json:"text"`
	ContextCfg          ContextConfig `json:"contextConfig"`
	LastUpdatedAt       int           `json:"lastUpdatedAt"`
	DisplayName         string        `json:"displayName"`
	Keys                []string      `json:"keys"`
	SearchRange         int           `json:"searchRange"`
	Enabled             bool          `json:"enabled"`
	ForceActivation     bool          `json:"forceActivation"`
	KeyRelative         bool          `json:"keyRelative"`
	NonStoryActivatable bool          `json:"nonStoryActivatable"`
	Tokens              []uint16
	KeysRegex           []*regexp.Regexp
}

type LorebookSettings struct {
	OrderByKeyLocations bool `json:"orderByKeyLocations"`
}

type Lorebook struct {
	Version  int              `json:"lorebookVersion"`
	Entries  []LorebookEntry  `json:"entries"`
	Settings LorebookSettings `json:"settings"`
}

type ScenarioSettings struct {
	Parameters    novelai_api.NaiGenerateParams
	TrimResponses bool `json:"trimResponses"`
	BanBrackets   bool `json:"banBrackets"`
}

type Scenario struct {
	ScenarioVersion int              `json:"scenarioVersion""`
	Title           string           `json:"title"`
	Author          string           `json:"author"`
	Description     string           `json:"description"`
	Prompt          string           `json:"prompt"`
	Tags            []string         `json:"tags"`
	Context         []ContextEntry   `json:"context"`
	Settings        ScenarioSettings `json:"settings"`
	Lorebook        Lorebook         `json:"lorebook"`
	Tokenizer       *gpt_bpe.GPTEncoder
}

func (scenario Scenario) ResolveLorebook(contexts ContextEntries) (entries ContextEntries) {
	for loreIdx := range scenario.Lorebook.Entries {
		lorebookEntry := scenario.Lorebook.Entries[loreIdx]
		if !lorebookEntry.Enabled {
			continue
		}
		keys := lorebookEntry.Keys
		keysRegex := lorebookEntry.KeysRegex
		indexes := make([][]int, 0)
		searchRange := lorebookEntry.SearchRange
		for keyIdx := range keysRegex {
			keyRegex := keysRegex[keyIdx]
			for ctxIdx := range contexts {
				searchText := contexts[ctxIdx].Text
				searchLen := len(searchText) - searchRange
				if searchLen > 0 {
					searchText = searchText[searchLen:]
				}
				ctxMatches := keyRegex.FindAllStringIndex(searchText, -1)
				if searchLen > 0 {
					for ctxMatchIdx := range ctxMatches {
						ctxMatches[ctxMatchIdx][0] = ctxMatches[ctxMatchIdx][0] + searchLen
						ctxMatches[ctxMatchIdx][1] = ctxMatches[ctxMatchIdx][1] + searchLen
					}
				}
				indexes = append(indexes, ctxMatches...)
			}
		}
		if len(indexes) > 0 {
			entry := ContextEntry{
				Text: lorebookEntry.Text,
				ContextCfg: lorebookEntry.ContextCfg,
				Tokens: lorebookEntry.Tokens,
				Label: lorebookEntry.DisplayName,
				Indexes: indexes,
			}
			entries = append(entries, entry)
			fmt.Printf("KEYS: %v @ %v\n", keys, indexes)
		}
	}
	return entries
}

/*
Create context list
Add story to context list
Add story context array to list
Add active lorebook entries to the context list
Add active ephemeral entries to the context list
Add cascading lorebook entries to the context list
Determine token lengths of each entry
Determine reserved tokens for each entry
Sort context list by insertion order
For each entry in the context list
    trim entry
    insert entry
    reduce reserved tokens
*/



func (scenario Scenario) GenerateContext(story string) (newContext string) {
	storyEntry := ContextEntry{
		Text: story,
		ContextCfg: ContextConfig{
			Prefix: "",
			Suffix: "",
			ReservedTokens: 512,
			InsertionPosition: -1,
			TokenBudget: 2048,
			BudgetPriority: 0,
			TrimDirection: "trimTop",
			InsertionType: "newline",
			MaximumTrimType: "sentence",
		},
		Tokens: scenario.Tokenizer.Encode(story),
		Label: "Story",
	}
	contexts := ContextEntries{storyEntry}
	lorebookContexts := scenario.ResolveLorebook(contexts)
	contexts = append(contexts, scenario.Context...)
	contexts = append(contexts, lorebookContexts...)
	budget := int(1024 - scenario.Settings.Parameters.MaxLength)
	sort.Sort(sort.Reverse(contexts))
	for ctxIdx := range(contexts) {
		budget = budget - len(contexts[ctxIdx].Tokens)
		fmt.Printf("PRIORITY: %4v RESERVED: %4v ACTUAL: %4v LEFT: %4v LABEL: %8v INSERTION_POS: %4v INSERTION_TYPE: %8v TRIM_TYPE: %8v TRIM_DIRECTION: %10v\n",
			contexts[ctxIdx].ContextCfg.BudgetPriority,
			contexts[ctxIdx].ContextCfg.ReservedTokens,
			len(contexts[ctxIdx].Tokens),
			budget,
			contexts[ctxIdx].Label,
			contexts[ctxIdx].ContextCfg.InsertionPosition,
			contexts[ctxIdx].ContextCfg.InsertionType,
			contexts[ctxIdx].ContextCfg.MaximumTrimType,
			contexts[ctxIdx].ContextCfg.TrimDirection)
	}
	return""
}

func ScenarioFromFile(tokenizer *gpt_bpe.GPTEncoder, path string) (scenario Scenario) {
	scenarioBytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(scenarioBytes, &scenario)
	if err != nil {
		log.Fatal(err)
	}
	scenario.Tokenizer = tokenizer
	for ctxIdx := range scenario.Context {
		ctx := scenario.Context[ctxIdx]
		ctx.Tokens = tokenizer.Encode(ctx.ContextCfg.Prefix +
			ctx.Text + ctx.ContextCfg.Suffix)
		scenario.Context[ctxIdx] = ctx
	}
	scenario.Context[0].Label = "Memory"
	scenario.Context[1].Label = "A/N"
	for loreIdx := range scenario.Lorebook.Entries {
		loreEntry := scenario.Lorebook.Entries[loreIdx]
		loreEntry.Tokens = tokenizer.Encode(loreEntry.Text)
		for keyIdx := range loreEntry.Keys {
			key := loreEntry.Keys[keyIdx]
			keyRegex, err := regexp.Compile("(?i)(^|\\W)(" + key + ")($|\\W)")
			if err != nil {
				log.Fatal(err)
			}
			loreEntry.KeysRegex = append(loreEntry.KeysRegex, keyRegex)
		}
		scenario.Lorebook.Entries[loreIdx] = loreEntry
	}

	return scenario
}
