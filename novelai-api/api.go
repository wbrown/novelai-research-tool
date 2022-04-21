package novelai_api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strings"

	"github.com/cenkalti/backoff/v4"
	"github.com/wbrown/gpt_bpe"
	"github.com/wbrown/novelai-research-tool/structs"
)

func GetEncoderByModel(id string) *gpt_bpe.GPTEncoder {
	if strings.HasPrefix(id, "krake") {
		return &gpt_bpe.PileEncoder
	} else {
		return &gpt_bpe.GPT2Encoder
	}
}

//
// Logprob structures
//

type LogprobPair struct {
	Before *float32
	After  *float32
}

type Logprob struct {
	Tokens   gpt_bpe.Tokens
	Logprobs LogprobPair
}

func (lp *LogprobPair) UnmarshalJSON(buf []byte) error {
	tmp := []interface{}{&lp.Before, &lp.After}
	wantLen := len(tmp)
	if err := json.Unmarshal(buf, &tmp); err != nil {
		return err
	}
	if g, e := len(tmp), wantLen; g != e {
		return fmt.Errorf(
			"wrong number of fields in LogprobPair: %d != %d", g, e)
	}
	return nil
}

func (lp *Logprob) MarshalJSON() ([]byte, error) {
	lpPair := []*float32{lp.Logprobs.Before, lp.Logprobs.After}
	lpTokens := lp.Tokens
	return json.Marshal([]interface{}{lpTokens, lpPair})
}

func (l *Logprob) UnmarshalJSON(buf []byte) error {
	tmp := []interface{}{&l.Tokens, &l.Logprobs}
	wantLen := len(tmp)
	if err := json.Unmarshal(buf, &tmp); err != nil {
		return err
	}
	if g, e := len(tmp), wantLen; g != e {
		return fmt.Errorf(
			"wrong number of fields in Logprob: %d != %d", g, e)
	}
	return nil
}

type LogprobEntry struct {
	Chosen *[]Logprob `json:"chosen"`
	Before *[]Logprob `json:"before"`
	After  *[]Logprob `json:"after"`
}

// Logit processing and order

type LogitProcessorID uint16

const (
	Temperature LogitProcessorID = iota
	Top_K
	Top_P
	TFS
)

type LogitProcessorIDs []LogitProcessorID
type LogitProcessorRepr string
type LogitProcessorReprMap map[LogitProcessorID]LogitProcessorRepr
type LogitProcessorReprs []LogitProcessorRepr

var LogitProcessorIdMap = LogitProcessorReprMap{
	Temperature: "Temperature",
	Top_K:       "Top_K",
	Top_P:       "Top_P",
	TFS:         "TFS",
}

func (lpids *LogitProcessorIDs) check() error {
	if len(*lpids) != len(LogitProcessorIdMap) {
		return errors.New("Must have four logit IDs in `order`!")
	}
	seen := make(LogitProcessorIDs, 0)
	for idIdx := range *lpids {
		for seenIdx := range seen {
			if seen[seenIdx] == (*lpids)[idIdx] {
				return errors.New(
					"Duplicate entry found in logit `order`!")
			}
		}
		seen = append(seen, (*lpids)[idIdx])
	}
	return nil
}

func (lpr *LogitProcessorRepr) toId() (LogitProcessorID, error) {
	currRepr := strings.Replace(strings.ToLower(string(*lpr)),
		"-", "_", -1)
	for lookupIdx := range LogitProcessorIdMap {
		if strings.ToLower(string(LogitProcessorIdMap[lookupIdx])) == currRepr {
			return lookupIdx, nil
		}
	}
	return 0, errors.New(fmt.Sprintf("Logit `%s` is not valid!", lpr))
}

func (lprs *LogitProcessorReprs) toIds() (*LogitProcessorIDs, error) {
	ids := make(LogitProcessorIDs, 0)
	for currReprIdx := range *lprs {
		if logitId, err := (*lprs)[currReprIdx].toId(); err != nil {
			return nil, err
		} else {
			ids = append(ids, logitId)
		}
	}
	if err := ids.check(); err != nil {
		return nil, err
	}
	return &ids, nil
}

func (id *LogitProcessorID) UnmarshalJSON(buf []byte) error {
	var tmp interface{}
	if err := json.Unmarshal(buf, &tmp); err != nil {
		return err
	}
	if newIntId, ok := tmp.(uint16); ok {
		logitId := interface{}(newIntId).(LogitProcessorID)
		id = &logitId
		return nil
	} else if repr, ok := tmp.(string); ok {
		logitRepr := LogitProcessorRepr(repr)
		if convId, err := logitRepr.toId(); err != nil {
			return err
		} else {
			newId := convId
			*id = newId
			return nil
		}
	} else {
		return errors.New("Logit ID is not a string or an uint!")
	}
}

func (ids *LogitProcessorIDs) UnmarshalJSON(buf []byte) error {
	var serIds []LogitProcessorID
	if err := json.Unmarshal(buf, &serIds); err == nil {
		newIds := LogitProcessorIDs(serIds)
		if err = newIds.check(); err != nil {
			return err
		} else {
			*ids = newIds
			return nil
		}
	} else {
		return err
	}
}

func (id LogitProcessorID) String() string {
	var repr LogitProcessorRepr
	var ok bool
	if repr, ok = LogitProcessorIdMap[id]; !ok {
		repr = "UNKNOWN"
	}
	return fmt.Sprintf("<%s>", repr)
}

//
// General NovelAI API structures
//

type NovelAiAPI struct {
	backend string
	keys    NaiKeys
	client  *http.Client
}

type NaiGenerateHTTPResp struct {
	Output     string          `json:"output"`
	Error      string          `json:"error"`
	StatusCode int             `json:"statusCode"`
	Message    string          `json:"message"`
	Logprobs   *[]LogprobEntry `json:"logprobs""`
}

type NextArray struct {
	Output [][]interface{} `json:"output"`
}

type NaiGenerateParams struct {
	Label                      *string             `json:"label,omitempty"`
	Model                      *string             `json:"model,omitempty"`
	Prefix                     *string             `json:"prefix,omitempty"`
	LogitMemory                *string             `json:"memory,omitempty"`
	LogitAuthors               *string             `json:"authorsnote,omitempty"`
	Temperature                *float64            `json:"temperature,omitempty"`
	MaxLength                  *uint               `json:"max_length,omitempty"`
	ContextLength              *uint               `json:"context_length,omitempty"`
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
	RepWhitelistIds            *[]uint16           `json:"repetition_penalty_whitelist,omitempty"`
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
}

type NaiGenerateResp struct {
	Request          string          `json:"request"`
	Response         string          `json:"response"`
	EncodedRequest   string          `json:"encoded_request"`
	EncodedResponse  string          `json:"encoded_response"`
	Logprobs         *[]LogprobEntry `json:"logprobs_response"`
	NextWordArray    [256][2]string
	NextWordReturned int
	Error            error `json:"error"`
}

func LogitBias() [][]float32 {
	return [][]float32{{0, 0.0}}
}

func RepWhitelistIds() []uint16 {
	return []uint16{0}
}

func EndOfTextTokens() [][]uint16 {
	return [][]uint16{{27, 91, 437, 1659, 5239, 91, 29},
		{1279, 91, 437, 1659, 5239, 91, 29},
		{27, 91, 10619, 46, 9792, 13918, 91, 29},
		{1279, 91, 10619, 46, 9792, 13918, 91, 29}}
}

func (params *NaiGenerateParams) CoerceNullValues(other *NaiGenerateParams) {
	if other == nil {
		return
	}
	if params == nil {
		*params = NaiGenerateParams{}
	}
	fields := reflect.TypeOf(*params)
	for field := 0; field < fields.NumField(); field++ {
		fieldValues := reflect.ValueOf(*params).Field(field)
		otherValues := reflect.ValueOf(*other).Field(field)
		if fieldValues.IsNil() && !otherValues.IsNil() {
			reflect.ValueOf(params).Elem().Field(
				field).Set(otherValues)
		}
	}
}

func (params *NaiGenerateParams) CoerceDefaults() {
	defaults := NewGenerateParams()
	params.CoerceNullValues(&defaults)
}

func NewGenerateParams() NaiGenerateParams {
	model := "6B-v4"
	prefix := "vanilla"
	temperature := 0.72
	maxLength := uint(40)
	contextLength := uint(2048)
	minLength := uint(1)
	topK := uint(0)
	topP := 0.725
	topA := 1.0
	typicalP := 1.0
	tfs := 1.0
	repPen := 3.5
	repPenRange := uint(2048)
	repPenSlope := 6.57
	repPenPresence := 0.0
	repPenFrequency := 0.0
	banBrackets := true
	badWordsIds := make([][]uint16, 0)
	logitBiasIds := make([][]float32, 0)
	repWhitelistIds := make([]uint16, 0)
	useCache := true
	useString := false
	returnFullText := false
	trimSpaces := true
	numLogprobs := uint(5)
	generateUntilSentence := true
	return NaiGenerateParams{
		Model:                      &model,
		Prefix:                     &prefix,
		Temperature:                &temperature,
		MaxLength:                  &maxLength,
		ContextLength:              &contextLength,
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
		Model:      *params.Model,
		Parameters: params,
	}
}

func generateGenRequest(encoded []byte, accessToken string,
	backendURI string) *http.Request {
	req, _ := http.NewRequest("POST", backendURI+"/ai/generate",
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
}

func (params NaiGenerateParams) GetScaledRepPen() float64 {
	const oldRange = 1 - 8.0
	const newRange = 1 - 1.525
	if *params.Model != "2.7B" {
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

func (api *NovelAiAPI) naiApiGenerate(params *NaiGenerateMsg) (
	respDecoded NaiGenerateHTTPResp) {

	params.Model = *params.Parameters.Model
	if *params.Parameters.BanBrackets {
		newBadWords := append(BannedBrackets(params.Model),
			*params.Parameters.BadWordsIds...)
		params.Parameters.BadWordsIds = &newBadWords
	}
	params.Parameters.ResolveRepetitionParams()
	params.Parameters.ResolveSamplingParams()

	if params.Parameters.BadWordsIds != nil && len(*params.Parameters.BadWordsIds) == 0 {
		params.Parameters.BadWordsIds = nil
	}
	if params.Parameters.LogitBiasIds != nil && len(*params.Parameters.LogitBiasIds) == 0 {
		params.Parameters.LogitBiasIds = nil
	}
	if params.Parameters.RepWhitelistIds != nil && len(*params.Parameters.RepWhitelistIds) == 0 {
		params.Parameters.RepWhitelistIds = nil
	}

	cl := http.DefaultClient
	encoded, _ := json.Marshal(params)
	req := generateGenRequest(encoded, api.keys.AccessToken, api.backend)
	// Retry up to 10 times.
	var resp *http.Response
	doGenerate := func() (err error) {
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
	if params.Parameters.NextWord == nil || *params.Parameters.NextWord == false {
		err = json.Unmarshal(body, &respDecoded)
		if err != nil {
			log.Printf("API: Error unmarshaling JSON response: %s %s",
				err, string(body))
			fmt.Scanln()
			os.Exit(1)
		}
		if len(respDecoded.Error) > 0 {
			log.Printf((fmt.Sprintf("API: Server error [%d]: %s",
				respDecoded.StatusCode, respDecoded.Error)))
			fmt.Scanln()
		}

	} else {
		respDecoded.Output = string(body)
	}
	return respDecoded
}

func NewNovelAiAPI() NovelAiAPI {
	auth := AuthEnv()
	return NovelAiAPI{
		backend: auth.Backend,
		keys:    auth,
		client:  http.DefaultClient,
	}
}

func (api *NovelAiAPI) GenerateWithParams(content *string,
	params NaiGenerateParams) (resp NaiGenerateResp) {
	if params.TrimSpaces == nil || *params.TrimSpaces == true {
		*content = strings.TrimRight(*content, " \t")
	}
	encoder := GetEncoderByModel(*params.Model)
	var val NextArray
	encoded := encoder.Encode(content)
	encodedBytes := encoded.ToBin()
	encodedBytes64 := base64.StdEncoding.EncodeToString(*encodedBytes)
	resp.Request = *content
	resp.EncodedRequest = encodedBytes64
	msg := NewGenerateMsg(encodedBytes64)
	msg.Parameters = params
	apiResp := api.naiApiGenerate(&msg)
	if params.NextWord == nil || *params.NextWord == false {
		if binTokens, err := base64.StdEncoding.DecodeString(apiResp.Output); err != nil {
			log.Println("ERROR:", err)
			resp.Error = err
		} else {
			tokens := gpt_bpe.TokensFromBin(&binTokens)
			/* if params.TrimResponses != nil && *params.TrimResponses == true {
				tokens, err = api.encoder.TrimIncompleteSentence(tokens)
			} */
			resp.Logprobs = apiResp.Logprobs
			resp.EncodedResponse = apiResp.Output
			resp.Response = encoder.Decode(tokens)
		}
	}

	if params.NextWord != nil && *params.NextWord == true {
		err := json.Unmarshal([]byte(apiResp.Output), &val)
		if err != nil {
			log.Printf("API: Error unmarshaling JSON NextWord response: %s %s",
				err, apiResp.Output)
			fmt.Scanln()
			os.Exit(1)
		}

		//decode next_word array
		nextArrayDecode := map[string]interface{}{}

		err = json.Unmarshal([]byte(apiResp.Output), &nextArrayDecode)
		if err != nil {
			fmt.Println(err)
		}
		get_keys := nextArrayDecode["output"]
		get_array := reflect.ValueOf(get_keys)
		//filter them into the NextWordArray
		for i := 0; i < get_array.Len(); i++ {
			getEntry := get_array.Index(i).Interface()
			next_ := reflect.ValueOf(getEntry)
			nextValueToken := next_.Index(0).Interface()
			nextValueWeight := next_.Index(1).Interface()

			//add to array
			(resp.NextWordArray)[i][0] = fmt.Sprintf("%v", nextValueToken)
			(resp.NextWordArray)[i][1] = fmt.Sprintf("%v", nextValueWeight)

			resp.NextWordReturned++
		}
	}
	return resp
}

func (api *NovelAiAPI) Generate(content string) (decoded string) {
	defaultParams := NewGenerateParams()
	return api.GenerateWithParams(&content, defaultParams).Response
}
