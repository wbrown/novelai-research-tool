package main

import (
	"fmt"
	nrt "github.com/wbrown/novelai-research-tool"
	"os"
	"path/filepath"
)

func main() {
	binName := filepath.Base(os.Args[0])

	if len(os.Args) != 2 {
		fmt.Printf("%v: %s dir/test.json\n", binName, os.Args[0])
		os.Exit(1)
	}
	inputPath := os.Args[1]
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Printf("%v: `%v` does not exist!\n", binName, inputPath)
		os.Exit(1)
	}
	tests := nrt.GenerateTestsFromFile(inputPath)
	fmt.Printf("== %v tests generated from %v ==\n", len(tests), inputPath)
	for testIdx := range tests {
		fmt.Printf("== Performing test %v / %v ==\n", testIdx+1, len(tests))
		tests[testIdx].Perform()
	}
}