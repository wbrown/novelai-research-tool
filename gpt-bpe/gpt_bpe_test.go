package gpt_bpe

import (
	"reflect"
	"testing"
)

var encoder GPTEncoder

// AssertEqual checks if values are equal
func AssertEqual(t *testing.T, a interface{}, b interface{}) {
	if reflect.DeepEqual(a, b) {
		return
	}
	t.Errorf("Received %v (type %v), expected %v (type %v)", a, reflect.TypeOf(a), b, reflect.TypeOf(b))
}

func TestMain(m *testing.M) {
	encoder = NewEncoder()
	m.Run()
}

func TestGPTEncoder_Split(t *testing.T) {
	AssertEqual(t,
		encoder.SplitWords("we'll go jump in a lake."),
		[]string{"we", "'ll", " go", " jump", " in", " a", " lake", "."})
	AssertEqual(t,
		encoder.SplitWords("multiple  encoded spaces."),
		[]string{"multiple", "  ", "encoded", " spaces", "."})
	AssertEqual(t,
		encoder.SplitWords("Capitalized Words Are Cool"),
		[]string{"Capitalized", " Words", " Are", " Cool"})
	AssertEqual(t,
		encoder.SplitWords("we'LL test irregular cApitalizatioN."),
		[]string{"we", "'LL", " test", " irregular", " cApitalizatioN", "."})
	AssertEqual(t,
		encoder.SplitWords(`multilines
are awesome`),
		[]string{"multilines", "\n", "are", " awesome"})
}

func TestGPTEncoder_Encode(t *testing.T) {
	// TBD
}

func TestGPTDecoder_Decode(t *testing.T) {
	// TBD
}

func TestRankPairs(t *testing.T) {
}