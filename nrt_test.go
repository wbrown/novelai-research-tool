package nrt

import (
	"testing"
)

func TestContentTest_GeneratePermutations(t *testing.T) {
	test := LoadSpecFromFile("tests/calliope.json")
	// Test behaviors when `2.7B` is the only `model` permutation value.
	tests := test.GeneratePermutations()
	if len(tests) != 2 {
		t.Error("tests/calliope.json should not be producing more than 2 permutation!")
	}
	if tests[1].Parameters.Model != "2.7B" {
		t.Error("2.7B model was not produced in the permutation output!")
	}
	if tests[1].Parameters.Prefix != "vanilla" {
		t.Error("2.7B model should only produce a `Prefix` of `vanilla`")
	}
	// Test behaviors when `6B-v3` is added to the permutation for `Model`.
	test.Permutations[0].Model = []string{"2.7B", "6B-v3"}
	tests = test.GeneratePermutations()
	if len(tests) != 3 {
		t.Error("tests/calliope.json with {\"model\":[\"2.7B\", \"6B-v3\"]} should be producing three permutations!")
	}
}