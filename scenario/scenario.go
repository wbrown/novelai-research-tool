package scenario

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
	Text         string        `json:"text"`
	ContextCfg   ContextConfig `json:"contextConfig"`
	Tokens       *[]uint16
	Label        string
	MatchIndexes []map[string][][]int
	Index        uint
}

type ContextEntries []ContextEntry

func (ctxes ContextEntries) Len() int {
	return len(ctxes)
}

func (ctxes ContextEntries) Swap(i, j int) {
	ctxes[i], ctxes[j] = ctxes[j], ctxes[i]
}

func (ctxes ContextEntries) Less(i, j int) bool {
	if ctxes[i].ContextCfg.BudgetPriority <
		ctxes[j].ContextCfg.BudgetPriority {
		return true
	} else if ctxes[i].ContextCfg.BudgetPriority ==
		ctxes[j].ContextCfg.BudgetPriority {
		return ctxes[i].Index > ctxes[j].Index
	}
	return false
}

type MaxTrimType uint

const (
	TrimSentences MaxTrimType = iota
	TrimNewlines  MaxTrimType = iota
	TrimTokens    MaxTrimType = iota
)

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
	Tokens              *[]uint16
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
	beginIdx := len(contexts)
	for loreIdx := range scenario.Lorebook.Entries {
		lorebookEntry := scenario.Lorebook.Entries[loreIdx]
		if !lorebookEntry.Enabled {
			continue
		}
		keys := lorebookEntry.Keys
		keysRegex := lorebookEntry.KeysRegex
		indexes := make([]map[string][][]int, 0)
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
				keyMatches := make(map[string][][]int, 0)
				if searchLen > 0 {
					for ctxMatchIdx := range ctxMatches {
						ctxMatches[ctxMatchIdx][0] = ctxMatches[ctxMatchIdx][0] + searchLen
						ctxMatches[ctxMatchIdx][1] = ctxMatches[ctxMatchIdx][1] + searchLen
					}
				}
				if len(ctxMatches) > 0 {
					keyMatches[keys[keyIdx]] = append(keyMatches[keys[keyIdx]], ctxMatches...)
				}
				if len(keyMatches) > 0 {
					indexes = append(indexes, keyMatches)
				}
			}
		}
		if len(indexes) > 0 || lorebookEntry.ForceActivation {
			entry := ContextEntry{
				Text: lorebookEntry.Text,
				ContextCfg: lorebookEntry.ContextCfg,
				Tokens: lorebookEntry.Tokens,
				Label: lorebookEntry.DisplayName,
				MatchIndexes: indexes,
				Index: uint(beginIdx + loreIdx),
			}
			entries = append(entries, entry)
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

func (scenario Scenario) createStoryContext(story string) ContextEntry {
	return ContextEntry{
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
		Tokens: scenario.Tokenizer.Encode(&story),
		Label: "Story",
		Index: 0,
	}
}

func getReservedContexts(ctxts ContextEntries) (reserved ContextEntries) {
	for ctxIdx := range(ctxts) {
		ctx := ctxts[ctxIdx]
		if ctx.ContextCfg.ReservedTokens > 0 {
			reserved = append(reserved, ctx)
		}
	}
	sort.Sort(sort.Reverse(reserved))
	return reserved
}

func (ctx *ContextEntry) getTrimDirection() gpt_bpe.TrimDirection {
	switch ctx.ContextCfg.TrimDirection {
	case "trimTop":
		return gpt_bpe.TrimTop
	case "trimBottom":
		return gpt_bpe.TrimBottom
	default:
		return gpt_bpe.TrimNone
	}
}

func (ctx *ContextEntry) getMaxTrimType() MaxTrimType {
	switch ctx.ContextCfg.MaximumTrimType {
	case "sentence":
		return TrimSentences
	case "newline":
		return TrimNewlines
	case "token":
		return TrimTokens
	default:
		return TrimNewlines
	}
}

func (ctx *ContextEntry) ResolveTrim(tokenizer *gpt_bpe.GPTEncoder, budget int) (trimmedTokens *[]uint16) {
	trimSize := 0
	numTokens := len(*ctx.Tokens)
	projected := budget - numTokens + ctx.ContextCfg.ReservedTokens
	if projected > ctx.ContextCfg.TokenBudget {
		trimSize = ctx.ContextCfg.TokenBudget
	} else if projected >= 0 {
		// We have enough to fit this into the budget.
		trimSize = numTokens
	} else {
		if float32(numTokens)*0.3 <= float32(budget) {
			trimSize = budget
		} else {
			trimSize = 0
		}
	}
	trimDirection := ctx.getTrimDirection()
	maxTrimType := ctx.getMaxTrimType()
	trimmedTokens, _ = tokenizer.TrimNewlines(ctx.Tokens, trimDirection, uint(trimSize))
	if len(*trimmedTokens) == 0 && maxTrimType >= TrimSentences {
		trimmedTokens, _ = tokenizer.TrimSentences(ctx.Tokens, trimDirection, uint(trimSize))
	}
	if len(*trimmedTokens) == 0 && maxTrimType == TrimTokens {
		tokens := *ctx.Tokens
		switch trimDirection {
		case gpt_bpe.TrimTop:
			tokens = tokens[numTokens-trimSize:]
		case gpt_bpe.TrimBottom:
			tokens = tokens[:trimSize]
		default:
			tokens = *trimmedTokens
		}
		trimmedTokens = &tokens
	}
	return trimmedTokens
}

func (scenario Scenario) GenerateContext(story string, budget int) (newContext string) {
	storyEntry := scenario.createStoryContext(story)
	contexts := ContextEntries{storyEntry}
	lorebookContexts := scenario.ResolveLorebook(contexts)
	contexts = append(contexts, scenario.Context...)
	contexts = append(contexts, lorebookContexts...)
	budget -= int(scenario.Settings.Parameters.MaxLength)
	reservedContexts := getReservedContexts(contexts)
	for ctxIdx := range reservedContexts {
		ctx := reservedContexts[ctxIdx]
		reservedTokens := ctx.ContextCfg.ReservedTokens
		szTokens := len(*ctx.Tokens)
		if szTokens < reservedTokens {
			budget -= szTokens
		} else {
			budget -= reservedTokens
		}
	}
	sort.Sort(sort.Reverse(contexts))
	for ctxIdx := range(contexts) {
		ctx := contexts[ctxIdx]
		trimmedTokens := ctx.ResolveTrim(scenario.Tokenizer, budget)
		numTokens := len(*trimmedTokens)
		budget -= numTokens
		fmt.Printf("PRIORITY: %4v RESERVED: %4v ACTUAL: %4v TRIMMED: %4v LEFT: %4v LABEL: %15v INSERTION_POS: %4v INSERTION_TYPE: %8v TRIM_TYPE: %8v TRIM_DIRECTION: %10v\n",
			contexts[ctxIdx].ContextCfg.BudgetPriority,
			contexts[ctxIdx].ContextCfg.ReservedTokens,
			len(*contexts[ctxIdx].Tokens),
			numTokens,
			budget,
			contexts[ctxIdx].Label,
			contexts[ctxIdx].ContextCfg.InsertionPosition,
			contexts[ctxIdx].ContextCfg.InsertionType,
			contexts[ctxIdx].ContextCfg.MaximumTrimType,
			contexts[ctxIdx].ContextCfg.TrimDirection)
	}
	return ""
}

func ScenarioFromFile(tokenizer *gpt_bpe.GPTEncoder, path string) (scenario Scenario, err error) {
	scenarioBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return scenario, err
	}
	err = json.Unmarshal(scenarioBytes, &scenario)
	if err != nil {
		return scenario, err
	}
	scenario.Tokenizer = tokenizer
	for ctxIdx := range scenario.Context {
		ctx := scenario.Context[ctxIdx]
		toEncode := ctx.ContextCfg.Prefix +
			ctx.Text + ctx.ContextCfg.Suffix
		ctx.Tokens = tokenizer.Encode(&toEncode)
		scenario.Context[ctxIdx] = ctx
	}
	scenario.Context[0].Label = "Memory"
	scenario.Context[0].Index = 1
	scenario.Context[1].Label = "A/N"
	scenario.Context[1].Index = 2
	for loreIdx := range scenario.Lorebook.Entries {
		loreEntry := scenario.Lorebook.Entries[loreIdx]
		loreEntry.Tokens = tokenizer.Encode(&loreEntry.Text)
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
	return scenario, err
}
