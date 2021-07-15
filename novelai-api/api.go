package novelai_api

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	gpt_bpe "github.com/wbrown/novelai-research-tool/gpt-bpe"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"
)

type NovelAiAPI struct {
	keys    NaiKeys
	client  *http.Client
	encoder gpt_bpe.GPTEncoder
}

func toBin(tokens []uint16) []byte {
	buf := bytes.NewBuffer(make([]byte, 0))
	for idx := range tokens {
		binary.Write(buf, binary.LittleEndian, tokens[idx])
	}
	return buf.Bytes()
}

func fromBin(bin []byte) (tokens []uint16) {
	buf := bytes.NewReader(bin)
	for {
		var token uint16
		if err := binary.Read(buf, binary.LittleEndian, &token); err != nil {
			break
		}
		tokens = append(tokens, token)
	}
	return tokens
}

type NaiGenerateHTTPResp struct {
	Output     string `json:"output"`
	Error      string `json:"error"`
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
}

type NaiGenerateParams struct {
	Label                  string     `json:"label"`
	Model                  string     `json:"model"`
	Prefix                 string     `json:"prefix"`
	PromptFilename         string     `json:"prompt_filename"`
	Temperature            float64    `json:"temperature"`
	MaxLength              uint       `json:"max_length"`
	MinLength              uint       `json:"min_length"`
	TopK                   uint       `json:"top_k"`
	TopP                   float64    `json:"top_p"`
	TailFreeSampling       float64    `json:"tail_free_sampling"`
	RepetitionPenalty      float64    `json:"repetition_penalty"`
	RepetitionPenaltyRange uint       `json:"repetition_penalty_range"`
	RepetitionPenaltySlope float64    `json:"repetition_penalty_slope"`
	BadWordsIds            [][]uint16 `json:"bad_words_ids"`
	BanBrackets            bool       `json:"ban_brackets"`
	UseCache               bool       `json:"use_cache"`
	UseString              bool       `json:"use_string"`
	ReturnFullText         bool       `json:"return_full_text"`
}

type NaiGenerateResp struct {
	EncodedRequest  string `json:"encoded_request"`
	EncodedResponse string `json:"encoded_response"`
	Response        string `json:"response"`
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

func NewGenerateParams() NaiGenerateParams {
	return NaiGenerateParams{
		Model:                  "6B-v3",
		Prefix:                 "vanilla",
		Temperature:            0.55,
		MaxLength:              40,
		MinLength:              40,
		TopK:                   140,
		TopP:                   0.9,
		TailFreeSampling:       1,
		RepetitionPenalty:      3.5,
		RepetitionPenaltyRange: 1024,
		RepetitionPenaltySlope: 6.57,
		BadWordsIds:            [][]uint16{},
		BanBrackets:            true,
		UseCache:               false,
		UseString:              false,
		ReturnFullText:         false,
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

func naiApiGenerate(keys NaiKeys, params NaiGenerateMsg) (respDecoded NaiGenerateHTTPResp) {
	params.Model = params.Parameters.Model
	const oldRange = 1 - 8.0
	const newRange = 1 - 1.525
	if params.Model != "2.7B" {
		params.Parameters.RepetitionPenalty =
			((params.Parameters.RepetitionPenalty-1)*newRange)/oldRange + 1
	}
	if params.Parameters.BanBrackets {
		params.Parameters.BadWordsIds = append(BannedBrackets(),
			params.Parameters.BadWordsIds...)
	}
	cl := http.DefaultClient
	encoded, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", "https://api.novelai.net/ai/generate",
		bytes.NewBuffer(encoded))
	req.Header.Set("User-Agent",
		"nrt/0.1 (" + runtime.GOOS + "; " + runtime.GOARCH +")")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+keys.AccessToken)

	// Retry up to 3 times.
	var resp *http.Response
	for tries := 0; tries < 3; tries++ {
		var err error
		resp, err = cl.Do(req)
		if err == nil {
			break
		}
		log.Println(err)
		time.Sleep(3)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("API: Error reading HTTP body: %s", err)
		os.Exit(1)
	}
	err = json.Unmarshal(body, &respDecoded)
	if err != nil {
		log.Printf("API: Error unmarshaling JSON response: %s", err)
		os.Exit(1)
	}
	if len(respDecoded.Error) > 0 {
		log.Fatal(fmt.Sprintf("API: Server error [%s]: %s",
			respDecoded.StatusCode, respDecoded.Error))
	}
	return respDecoded
}

func NewNovelAiAPI() NovelAiAPI {
	return NovelAiAPI{
		keys:    AuthEnv(),
		client:  http.DefaultClient,
		encoder: gpt_bpe.NewEncoder(),
	}
}

func (api NovelAiAPI) GenerateWithParams(content string, params NaiGenerateParams) (resp NaiGenerateResp) {
	encoded := api.encoder.Encode(content)
	encodedBytes := toBin(encoded)
	encodedBytes64 := base64.StdEncoding.EncodeToString(encodedBytes)
	resp.EncodedRequest = encodedBytes64
	msg := NewGenerateMsg(encodedBytes64)
	msg.Parameters = params
	apiResp := naiApiGenerate(api.keys, msg)
	if binTokens, err := base64.StdEncoding.DecodeString(apiResp.Output); err != nil {
		log.Println("ERROR:", err)
		resp.Error = err
	} else {
		resp.EncodedResponse = apiResp.Output
		resp.Response = api.encoder.Decode(fromBin(binTokens))
	}
	return resp
}

func (api NovelAiAPI) Generate(content string) (decoded string) {
	defaultParams := NewGenerateParams()
	return api.GenerateWithParams(content, defaultParams).Response
}
