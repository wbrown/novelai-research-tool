package main

import (
	"fmt"
	nrt "github.com/wbrown/novelai-research-tool"
	"os"
	"path/filepath"
	"sync"
)

func threadWorker(wg *sync.WaitGroup, tests *chan nrt.ContentTest, total int) {
	defer wg.Done()
	for test := range *tests {
		testIdx := test.Index
		fmt.Printf("== Performing test %v / %v ==\n", testIdx, total)
		test.Perform()
	}
	wg.Done()
}

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
	workToDo := make(chan nrt.ContentTest, 1)
	var wg sync.WaitGroup
	for idx := 0; idx < 1; idx++ {
		fmt.Println("nrt: Starting worker", idx)
		wg.Add(1)
		go threadWorker(&wg, &workToDo, len(tests))
	}
	for testIdx := range tests {
		tests[testIdx].Index = testIdx
		workToDo <- tests[testIdx]
	}
	wg.Wait()
}
