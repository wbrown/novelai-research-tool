package novelai_api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	gpt_bpe "github.com/wbrown/novelai-research-tool/gpt-bpe"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
)

type NovelAiAPI struct {
	backend string
	keys    NaiKeys
	client  *http.Client
	encoder gpt_bpe.GPTEncoder
}

type NaiGenerateHTTPResp struct {
	Output     string `json:"output"`
	Error      string `json:"error"`
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
}

type NaiGenerateParams struct {
<<<<<<< Updated upstream
	Label                  string      `json:"label"`
	Model                  string      `json:"model"`
	Prefix                 string      `json:"prefix"`
	Temperature            *float64    `json:"temperature"`
	MaxLength              *uint       `json:"max_length"`
	MinLength              *uint       `json:"min_length"`
	TopK                   *uint       `json:"top_k"`
	TopP                   *float64    `json:"top_p"`
	TopA                   *float64    `json:"top_a"`
	TailFreeSampling       *float64    `json:"tail_free_sampling"`
	RepetitionPenalty      *float64    `json:"repetition_penalty"`
	RepetitionPenaltyRange *uint       `json:"repetition_penalty_range"`
	RepetitionPenaltySlope *float64    `json:"repetition_penalty_slope"`
	BadWordsIds            *[][]uint16 `json:"bad_words_ids"`
	BanBrackets            *bool       `json:"ban_brackets"`
	UseCache               bool        `json:"use_cache"`
	UseString              bool        `json:"use_string"`
	ReturnFullText         bool        `json:"return_full_text"`
	TrimResponses          *bool        `json:"trim_responses"`
	TrimSpaces             *bool        `json:"trim_spaces"`
=======
	Label                      *string             `json:"label,omitempty"`
	Model                      *string             `json:"model,omitempty"`
	Prefix                     *string             `json:"prefix,omitempty"`
	LogitMemory                *string             `json:"memory,omitempty"`
	LogitAuthors               *string             `json:"authorsnote,omitempty"`
	Temperature                *float64            `json:"temperature,omitempty"`
	MaxLength                  *uint               `json:"max_length,omitempty"`
	MinLength                  *uint               `json:"min_length,omitempty"`
	TopK                       *uint               `json:"top_k,omitempty"`
	TopP                       *float64            `json:"top_p,omitempty"`
	TopA                       *float64            `json:"top_a,omitempty"`
	TypicalP                   *float64            `json:"typical_p,omitempty"`	
	TailFreeSampling           *float64            `json:"tail_free_sampling,omitempty"`
	RepetitionPenalty          *float64            `json:"repetition_penalty,omitempty"`
	RepetitionPenaltyRange     *uint               `json:"repetition_penalty_range,omitempty"`
	RepetitionPenaltySlope     *float64            `json:"repetition_penalty_slope,omitempty"`
	RepetitionPenaltyFrequency *float64            `json:"repetition_penalty_frequency,omitempty"`
	RepetitionPenaltyPresence  *float64            `json:"repetition_penalty_presence,omitempty"`
	RepWhitelistIds            *[]uint16         `json:"repetition_penalty_whitelist,omitempty"`
	BadWordsIds                *[][]uint16         `json:"bad_words_ids,omitempty"`
	LogitBiasIds               *[][]float32        `json:"logit_bias,omitempty"`
	LogitBiasGroups            *structs.BiasGroups `json:"logit_bias_groups,omitempty"`
	BanBrackets                *bool               `json:"ban_brackets,omitempty"`
	UseCache                   *bool               `json:"use_cache,omitempty"`
	UseString                  *bool               `json:"use_string,omitempty"`
	ReturnFullText             *bool               `json:"return_full_text,omitempty"`
	TrimSpaces                 *bool               `json:"trim_spaces,omitempty"`
	NonZeroProbs               *bool               `json:"output_nonzero_probs,omitempty"`
	NextWord                   *bool               `json:"next_word,omitempty"`
	NumLogprobs                *uint               `json:"num_logprobs,omitempty"`
	GenerateUntilSentence      *bool               `json:"generate_until_sentence"`
	Order                      *LogitProcessorIDs  `json:"order"`
>>>>>>> Stashed changes
}

type NaiGenerateResp struct {
	Request         string `json:"request"`
	Response        string `json:"response"`
	EncodedRequest  string `json:"encoded_request"`
	EncodedResponse string `json:"encoded_response"`
	Error           error  `json:"error"`
}

func BannedBrackets() [][]uint16 {
	return [][]uint16{{58}, {60}, {90}, {92}, {685}, {1391}, {1782},
		{2361}, {3693}, {4083}, {4357}, {4895}, {5512}, {5974}, {7131},
		{8183}, {8351}, {8762}, {8964}, {8973}, {9063}, {11208}, {11709},
		{11907}, {11919}, {12878}, {12962}, {13018}, {13412}, {14631},
		{14692}, {14980}, {15090}, {15437}, {16151}, {16410}, {16589},
		{17241}, {17414}, {17635}, {17816}, {17912}, {18083}, {18161},
		{18477}, {19629}, {19779}, {19953}, {20520}, {20598}, {20662},
		{20740}, {21476}, {21737}, {22133}, {22241}, {22345}, {22935},
		{23330}, {23785}, {23834}, {23884}, {25295}, {25597}, {25719},
		{25787}, {25915}, {26076}, {26358}, {26398}, {26894}, {26933},
		{27007}, {27422}, {28013}, {29164}, {29225}, {29342}, {29565},
		{29795}, {30072}, {30109}, {30138}, {30866}, {31161}, {31478},
		{32092}, {32239}, {32509}, {33116}, {33250}, {33761}, {34171},
		{34758}, {34949}, {35944}, {36338}, {36463}, {36563}, {36786},
		{36796}, {36937}, {37250}, {37913}, {37981}, {38165}, {38362},
		{38381}, {38430}, {38892}, {39850}, {39893}, {41832}, {41888},
		{42535}, {42669}, {42785}, {42924}, {43839}, {44438}, {44587},
		{44926}, {45144}, {45297}, {46110}, {46570}, {46581}, {46956},
		{47175}, {47182}, {47527}, {47715}, {48600}, {48683}, {48688},
		{48874}, {48999}, {49074}, {49082}, {49146}, {49946}, {10221},
		{4841}, {1427}, {2602, 834}, {29343}, {37405}, {35780}, {2602},
		{17202}, {8162}}
}

<<<<<<< Updated upstream
=======
func LogitBias() [][]float32 {
	return [][]float32{{50256, 0.0}}
}

func RepWhitelistIds() []uint16 {
	return []uint16{50256}
}


>>>>>>> Stashed changes
func EndOfTextTokens() [][]uint16 {
	return [][]uint16{{27,91,437,1659,5239,91,29},
		{1279,91,437,1659,5239,91,29},
		{27,91,10619,46,9792,13918,91,29},
		{1279,91,10619,46,9792,13918,91,29}}
}

func (params *NaiGenerateParams) CoerceNullValues(other NaiGenerateParams) {
	if params.Label == "" {
		params.Label = other.Label
	}
	if params.Model == "" {
		params.Model = other.Model
	}
	if params.Prefix == "" {
		params.Prefix = other.Prefix
	}
	if params.Temperature == nil {
		params.Temperature = other.Temperature
	}
	if params.MaxLength == nil {
		params.MaxLength = other.MaxLength
	}
	if params.MinLength == nil {
		params.MinLength = other.MinLength
	}
	if params.TopA == nil {
		params.TopA = other.TopA
	}
	if params.TopK == nil {
		params.TopK = other.TopK
	}
	if params.TopP == nil {
		params.TopP = other.TopP
	}
	if params.TailFreeSampling == nil {
		params.TailFreeSampling = other.TailFreeSampling
	}
	if params.RepetitionPenalty == nil {
		params.RepetitionPenalty = other.RepetitionPenalty
	}
	if params.RepetitionPenaltyRange == nil {
		params.RepetitionPenaltyRange = other.RepetitionPenaltyRange
	}
	if params.RepetitionPenaltySlope == nil {
		params.RepetitionPenaltySlope = other.RepetitionPenaltySlope
	}
	if params.BanBrackets == nil {
		params.BanBrackets = other.BanBrackets
	}
	if params.BadWordsIds == nil {
		params.BadWordsIds = other.BadWordsIds
	}
	if params.TrimSpaces == nil {
		params.TrimSpaces = other.TrimSpaces
	}
}

func (params *NaiGenerateParams) CoerceDefaults() {
	defaults := NewGenerateParams()
	params.CoerceNullValues(defaults)
}

func NewGenerateParams() NaiGenerateParams {
	temperature := 0.72
	maxLength := uint(40)
	minLength := uint(1)
	topK := uint(0)
	topP := 0.725
	tfs := 1.0
	repPen := 3.5
	repPenRange := uint(1024)
	repPenSlope := 6.57
	banBrackets := true
	badWordsIds := make([][]uint16, 0)
<<<<<<< Updated upstream
=======
	repWhitelistIds := make([]uint16, 0)
	logitBiasIds := make([][]float32, 0)
	useCache := true
	useString := false
	returnFullText := false
>>>>>>> Stashed changes
	trimSpaces := true
	return NaiGenerateParams{
<<<<<<< Updated upstream
		Model:                  "6B-v3",
		Prefix:                 "general_crossgenre",
		Temperature:            &temperature,
		MaxLength:              &maxLength,
		MinLength:              &minLength,
		TopK:                   &topK,
		TopP:                   &topP,
		TailFreeSampling:       &tfs,
		RepetitionPenalty:      &repPen,
		RepetitionPenaltyRange: &repPenRange,
		RepetitionPenaltySlope: &repPenSlope,
		BadWordsIds:            &badWordsIds,
		BanBrackets:            &banBrackets,
		UseCache:               false,
		UseString:              false,
		ReturnFullText:         false,
		TrimSpaces:             &trimSpaces,
=======
		Model:                      &model,
		Prefix:                     &prefix,
		Temperature:                &temperature,
		MaxLength:                  &maxLength,
		MinLength:                  &minLength,
		TopA:                       &topA,
		TopK:                       &topK,
		TopP:                       &topP,
		TypicalP:                   &typicalP,
		TailFreeSampling:           &tfs,
		RepetitionPenalty:          &repPen,
		RepetitionPenaltyRange:     &repPenRange,
		RepetitionPenaltySlope:     &repPenSlope,
		RepetitionPenaltyPresence:  &repPenPresence,
		RepetitionPenaltyFrequency: &repPenFrequency,
		RepWhitelistIds:            &repWhitelistIds,
		BadWordsIds:                &badWordsIds,
		LogitBiasIds:               &logitBiasIds,
		BanBrackets:                &banBrackets,
		UseCache:                   &useCache,
		UseString:                  &useString,
		ReturnFullText:             &returnFullText,
		GenerateUntilSentence:      &generateUntilSentence,
		TrimSpaces:                 &trimSpaces,
		NumLogprobs:                &numLogprobs,
>>>>>>> Stashed changes
	}
}

type NaiGenerateMsg struct {
	Input      string            `json:"input"`
	Model      string            `json:"model"`
	Parameters NaiGenerateParams `json:"parameters"`
}

func NewGenerateMsg(input string) NaiGenerateMsg {
	params := NewGenerateParams()
	return NaiGenerateMsg{
		Input:      input,
		Model:      params.Model,
		Parameters: params,
	}
}

func generateGenRequest(encoded []byte, accessToken string, backendURI string) *http.Request {
	req, _ := http.NewRequest("POST", backendURI + "/ai/generate",
		bytes.NewBuffer(encoded))
	req.Header.Set("User-Agent",
		"nrt/0.1 ("+runtime.GOOS+"; "+runtime.GOARCH+")")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	return req
}

func (params *NaiGenerateParams) ResolveSamplingParams() {
	if params.TopP == nil || *params.TopP == 0 {
		topP := 1.0
		params.TopP = &topP
	}
	if params.TailFreeSampling == nil || *params.TailFreeSampling == 0 {
		tailFreeSampling := 1.0
		params.TailFreeSampling = &tailFreeSampling
	}
<<<<<<< Updated upstream
=======
	if params.TopA == nil || *params.TopA == 0 {
		topA := 1.0
		params.TopA = &topA
	}
	if params.TypicalP == nil || *params.TypicalP == 0 {
		typicalP := 1.0
		params.TypicalP = &typicalP
	}
>>>>>>> Stashed changes
}

func (params NaiGenerateParams) GetScaledRepPen() float64 {
	const oldRange = 1 - 8.0
	const newRange = 1 - 1.525
	if params.Model != "2.7B" {
		scaledRepPen := ((*params.RepetitionPenalty-1)*newRange)/oldRange + 1
		return scaledRepPen
	}
	return *params.RepetitionPenalty
}

func (params *NaiGenerateParams) ResolveRepetitionParams() {
	scaledRepPen := params.GetScaledRepPen()
	params.RepetitionPenalty = &scaledRepPen
	if params.RepetitionPenaltySlope != nil &&
		*params.RepetitionPenaltySlope == 0 {
		params.RepetitionPenaltySlope = nil
	}
	if params.RepetitionPenaltyRange != nil &&
		*params.RepetitionPenaltyRange == 0 {
		params.RepetitionPenaltyRange = nil
	}
}

func naiApiGenerate(keys *NaiKeys, params NaiGenerateMsg, backend string) (respDecoded NaiGenerateHTTPResp) {
	params.Model = params.Parameters.Model
	if *params.Parameters.BanBrackets {
		newBadWords := append(BannedBrackets(),
			*params.Parameters.BadWordsIds...)
		params.Parameters.BadWordsIds = &newBadWords
	}
	params.Parameters.ResolveRepetitionParams()
	params.Parameters.ResolveSamplingParams()
	if len(*params.Parameters.BadWordsIds) == 0 {
		params.Parameters.BadWordsIds = nil
	}
<<<<<<< Updated upstream
=======
	if len(*params.Parameters.LogitBiasIds) == 0 {
		params.Parameters.LogitBiasIds = nil
	}
	

	if *params.Parameters.RepWhitelistIds == nil {
	
	println(*params.Parameters.RepWhitelistIds)
	fmt.Scanln()
		temp_map := make([]uint16, 0)
		params.Parameters.RepWhitelistIds = &temp_map
	}
>>>>>>> Stashed changes
	cl := http.DefaultClient
	encoded, _ := json.Marshal(params)
	req := generateGenRequest(encoded, keys.AccessToken, backend)
	// Retry up to 10 times.
	var resp *http.Response
	doGenerate := func () (err error) {
		resp, err = cl.Do(req)
		if err == nil && resp.StatusCode == 201 {
			return err
		} else if resp != nil {
			body, readErr := ioutil.ReadAll(resp.Body)
			if readErr != nil {
				log.Printf("API: Error reading HTTP body of StatusCode: %d, %s\n",
					resp.StatusCode, readErr)
				return readErr
			}
			errStr := fmt.Sprintf("API: StatusCode: %d, %v, %v\n",
				resp.StatusCode, err, string(body))
			log.Print(errStr)
			return errors.New(errStr)
		} else {
			log.Printf("API: Error: %v\n", err)
		}
		return err
	}
	err := backoff.Retry(doGenerate, backoff.NewExponentialBackOff())
	if err != nil {
		log.Printf("API: Error: %v", err)
		os.Exit(1)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("API: Error reading HTTP body: %s", err)
		os.Exit(1)
	}
	err = json.Unmarshal(body, &respDecoded)
	if err != nil {
		log.Printf("API: Error unmarshaling JSON response: %s %s", err, string(body))
		os.Exit(1)
	}
	if len(respDecoded.Error) > 0 {
		log.Fatal(fmt.Sprintf("API: Server error [%d]: %s",
			respDecoded.StatusCode, respDecoded.Error))
	}
	return respDecoded
}

func NewNovelAiAPI() NovelAiAPI {
	auth := AuthEnv()
	return NovelAiAPI{
		backend: auth.Backend,
		keys: auth,
		client: http.DefaultClient,
		encoder: gpt_bpe.NewEncoder(),
	}
}

func (api NovelAiAPI) GenerateWithParams(content *string, params NaiGenerateParams) (resp NaiGenerateResp) {
	if params.TrimSpaces == nil || *params.TrimSpaces == true {
		*content = strings.TrimRight(*content, " \t")
	}
	encoded := api.encoder.Encode(content)
	encodedBytes := encoded.ToBin()
	encodedBytes64 := base64.StdEncoding.EncodeToString(*encodedBytes)
	resp.Request = *content
	resp.EncodedRequest = encodedBytes64
	msg := NewGenerateMsg(encodedBytes64)
	msg.Parameters = params
	apiResp := naiApiGenerate(&api.keys, msg, api.backend)
	if binTokens, err := base64.StdEncoding.DecodeString(apiResp.Output); err != nil {
		log.Println("ERROR:", err)
		resp.Error = err
	} else {
		tokens := gpt_bpe.TokensFromBin(&binTokens)
		if params.TrimResponses != nil && *params.TrimResponses == true {
			tokens, err = api.encoder.TrimIncompleteSentence(tokens)
		}
		resp.EncodedResponse = apiResp.Output
		resp.Response = api.encoder.Decode(tokens)
	}
	return resp
}

func (api NovelAiAPI) Generate(content string) (decoded string) {
	defaultParams := NewGenerateParams()
	return api.GenerateWithParams(&content, defaultParams).Response
}
