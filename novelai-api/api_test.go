package novelai_api

import "testing"

type RepPenTest struct {
	input  float64
	output float64
}

var RepPenTests = []RepPenTest{
	{3.5, 1.187500},
	{0, 0.925000},
}

func TestNaiGenerateMsg_GetScaledRepPen(t *testing.T) {
	params := NewGenerateParams()
	for testIdx := range RepPenTests {
		test := RepPenTests[testIdx]
		*params.RepetitionPenalty = test.input
		output := params.GetScaledRepPen()
		if output != test.output {
			t.Errorf("GetScaledRepPen: expected %f, got %f",
				test.output, output)
		}
	}
}
