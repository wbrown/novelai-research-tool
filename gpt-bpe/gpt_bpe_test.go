package gpt_bpe

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"testing"
	"time"
)

var encoder GPTEncoder
var corpus string
var encoded *[]uint16

// AssertEqual checks if values are equal
func AssertEqual(t *testing.T, a interface{}, b interface{}) {
	if reflect.DeepEqual(a, b) {
		return
	}
	t.Errorf("Received %v (type %v), expected %v (type %v)", a, reflect.TypeOf(a), b, reflect.TypeOf(b))
}

func TestMain(m *testing.M) {
	encoder = NewEncoder()
	if textBytes, err := os.ReadFile("resources/frankenstein.txt"); err != nil {
		log.Fatal("Error opening `resources/frankenstein.txt`")
	} else {
		corpus = string(textBytes)
	}
	m.Run()
}

type TrimTest struct {
	Input     string
	Direction TrimDirection
	Limit     uint
	Expected  string
}

const sent1 = "This is test sentence 1.  This is test sentence 2.  This is test sentence 3."
const sent2 = "This is test sentence 4.\nThis is test sentence 5.\nThis is test sentence 6."

var TrimTests = []TrimTest{
	{sent1, TrimTop, 10,
		" This is test sentence 3."},
	{sent1, TrimTop, 20,
		" This is test sentence 2.  This is test sentence 3."},
	{sent1, TrimTop, 30,
		sent1},
	{sent2, TrimTop, 10,
		"\nThis is test sentence 6."},
	{sent2, TrimTop, 18,
		"\nThis is test sentence 5.\nThis is test sentence 6."},
	{sent2, TrimTop, 30,
		sent2},
	{sent1, TrimBottom, 10,
		"This is test sentence 1. "},
	{sent1, TrimBottom, 20,
		"This is test sentence 1.  This is test sentence 2."},
	{sent1, TrimBottom, 30,
		sent1},
	{sent2, TrimBottom, 10,
		"This is test sentence 4.\n"},
	{sent2, TrimBottom, 18,
		"This is test sentence 4.\nThis is test sentence 5.\n"},
	{sent2, TrimBottom, 30,
		sent2},
}

func TestGPTEncoder_TrimSentences(t *testing.T) {
	for testIdx := range TrimTests {
		test := TrimTests[testIdx]
		res, err := encoder.TrimSentences(encoder.Encode(&test.Input),
			test.Direction, test.Limit)
		if err != nil {
			t.Error("TrimSentences: error:", err)
		}
		decodeRes := encoder.Decode(res)
		if decodeRes != test.Expected {
			t.Error("TrimSentences: expected '" + test.Expected + "' got '" +
				decodeRes + "'")
		}
	}
}

type SplitTest struct {
	Input     string
	Expected  []string
}

var SplitTests = []SplitTest{
	{"we'll go jump in a lake.",
		[]string{"we", "'ll", " go", " jump", " in", " a", " lake", "."}},
	{"multiple  encoded spaces.",
	[]string{"multiple", "  ", "encoded", " spaces", "."}},
	{"Capitalized Words Are Cool",
		[]string{"Capitalized", " Words", " Are", " Cool"}},
	{"we'LL test irregular cApitalizatioN.",
	[]string{"we", "'", "LL", " test", " irregular", " cApitalizatioN", "."}},
	{"multilines\nare awesome",
	[]string{"multilines", "\n", "are", " awesome"}}}

func TestGPTEncoder_Split(t *testing.T) {
	for testIdx := range(SplitTests) {
		test := SplitTests[testIdx]
		AssertEqual(t, *(encoder.SplitWords(&test.Input)), test.Expected)
	}
}

func BenchmarkGPTEncoder_Decode(b *testing.B) {
	if encoded == nil {
		corpEncoded := encoder.Encode(&corpus)
		encoded = corpEncoded
	}
	start := time.Now()
	tokenNumBytes := len(encoder.Decode(encoded))
	duration := time.Since(start)
	b.Log(fmt.Sprintf("%v tokens into %v bytes over %v",
		len(*encoded), tokenNumBytes, duration))
}

func BenchmarkGPTEncoder_Encode(b *testing.B) {
	start := time.Now()
	tokenCt := len(*encoder.Encode(&corpus))
	duration := time.Since(start)
	b.Log(fmt.Sprintf("%v bytes into %v tokens over %v",
		len(corpus), tokenCt, duration))
}

func TestGPTEncoder_Encode(t *testing.T) {
	start := time.Now()
	tokenCt := len(*encoder.Encode(&corpus))
	duration := time.Since(start)
	t.Log(fmt.Sprintf("%v bytes into %v tokens over %v\n",
		len(corpus), tokenCt, duration))
	// TBD
}

func TestGPTEncoder_Decode(t *testing.T) {
	if encoded == nil {
		corpEncoded := encoder.Encode(&corpus)
		encoded = corpEncoded
	}
	start := time.Now()
	tokenNumBytes := len(encoder.Decode(encoded))
	duration := time.Since(start)
	t.Log(fmt.Sprintf("%v tokens into %v bytes over %v\n",
		len(*encoded), tokenNumBytes, duration))
}

func TestGPTDecoder_Decode(t *testing.T) {
	// TBD
}

func TestRankPairs(t *testing.T) {
}
