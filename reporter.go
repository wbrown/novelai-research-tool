package nrt

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

func handleWrite(f *os.File, s string) {
	_, err := f.WriteString(s)
	if err != nil {
		log.Printf("reporter: Error writing string: `%s`: %s", s, err)
		os.Exit(1)
	}
}

//
// ConsoleReporter - reports on test progress to the console for the user
//

type ConsoleReporter struct {
	blue         func(a ...interface{}) string
	green        func(a ...interface{}) string
	blueNewline  string
	greenNewline string
	ct           *ContentTest
}

func (ct *ContentTest) CreateConsoleReporter() (ur ConsoleReporter) {
	ur.ct = ct
	ur.blue = color.New(color.FgWhite, color.BgBlue).SprintFunc()
	ur.blueNewline = ur.blue("\\n") + "\n"
	ur.green = color.New(color.FgWhite, color.BgGreen).SprintFunc()
	ur.greenNewline = ur.green("\\n") + "\n"
	fields := reflect.TypeOf(ct.Parameters)
	fmt.Printf("%v\n", ur.blue("Parameters:"))
	for field := 0; field < fields.NumField(); field++ {
		fieldName := fields.FieldByIndex([]int{field}).Name
		fieldValues := reflect.ValueOf(ct.Parameters).Field(field)
		if fieldValues.Kind() == reflect.Ptr {
			fieldValues = fieldValues.Elem()
		}
		fmt.Printf("%25v: %v\n", fieldName, fieldValues)
	}
	fmt.Printf("%v\n", ur.blue("Placeholders:"))
	for _, v := range ct.Scenario.PlaceholderMap {
		fmt.Printf("%25s: \"%s\"\n", v.Variable, v.Value)
	}

	return ur
}

func (cr *ConsoleReporter) ReportIteration(iteration int) {
	fmt.Printf("%v %v / %v\n", cr.blue("Iteration:"), iteration+1, cr.ct.Iterations)
	fmt.Printf("%v %v\n", cr.blue("Prompt:"),
		strings.Replace(cr.ct.Prompt, "\n", cr.blueNewline, -1))
	if len(cr.ct.Memory) > 0 {
		fmt.Printf("%v %v\n", cr.blue("Memory:"),
			strings.Replace(cr.ct.Memory, "\n", cr.blueNewline, -1))
	}
	if len(cr.ct.AuthorsNote) > 0 {
		fmt.Printf("%v %v\n", cr.blue("Author's Note:"),
			strings.Replace(cr.ct.AuthorsNote, "\n", cr.blueNewline, -1))
	}
}

func (cr *ConsoleReporter) ReportGeneration(resp string) {
	fmt.Printf("%v%v\n", cr.green("=>"),
		strings.Replace(resp, "\n", cr.greenNewline, -1))
}

func (cr *ConsoleReporter) close() {
	fmt.Printf("%v\n", cr.blue("== Test Instance Complete =="))
}

//
// JSONReporter - takes output of test iterations and serializes to a JSON
//                file
//

type JSONReporter struct {
	fileHandle *os.File
	iteration  int
}

func CreateJSONReporter(path string) (reportWriter JSONReporter) {
	var err error
	dir := filepath.Dir(path)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		log.Printf("reporter: Cannot create path: `%s`: %s", dir, err)
		os.Exit(1)
	}
	reportWriter.fileHandle, err = os.OpenFile(path,
		os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("reporter: Cannot open file for writing: `%s`: %s", path, err)
		os.Exit(1)
	}
	reportWriter.iteration = 0
	handleWrite(reportWriter.fileHandle, "[")
	return reportWriter
}

func (reportWriter *JSONReporter) SerializeIteration(result *IterationResult) {
	if reportWriter.iteration != 0 {
		handleWrite(reportWriter.fileHandle, ",\n")
	}
	reportWriter.iteration += 1
	serialized, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Printf("reporter: Cannot marshal JSON: %v, %v", result, err)
		os.Exit(1)
	}
	handleWrite(reportWriter.fileHandle, string(serialized))
	reportWriter.fileHandle.Sync()
}

func (reportWriter *JSONReporter) close() {
	handleWrite(reportWriter.fileHandle, "]")
	reportWriter.fileHandle.Close()
}

//
// TextReporter - takes output of test iterations and serializes to a JSON
//                file
//

type TextReporter struct {
	fileHandle *os.File
	ct         *ContentTest
}

func (ct ContentTest) CreateTextReporter(path string) (textReporter TextReporter) {
	var err error
	dir := filepath.Dir(path)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		log.Printf("reporter: Cannot create path: `%s`: %s", dir, err)
		os.Exit(1)
	}
	textReporter.fileHandle, err = os.OpenFile(path,
		os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("reporter: Cannot open file for writing: `%s`: %s", path, err)
		os.Exit(1)
	}
	paramsReportBytes, err := json.MarshalIndent(ct.Parameters, "", "")
	if err != nil {
		log.Printf("reporter: Cannot marshal JSON: %v, %v", ct.Parameters, err)
		os.Exit(1)
	}
	phReport := ""
	for _, v := range ct.Scenario.PlaceholderMap {
		phReport += fmt.Sprintf("%s:\"%s\"\n", v.Variable, v.Value)
	}
	replacer := strings.NewReplacer(
		"{\n", "",
		"}", "",
		",", "",
		"\"", "",
	)
	paramsReport := replacer.Replace(string(paramsReportBytes))
	handleWrite(textReporter.fileHandle, "=== Parameters ====================================\n")
	handleWrite(textReporter.fileHandle, paramsReport+"\n")
	handleWrite(textReporter.fileHandle, "=== Placeholders ==================================\n")
	handleWrite(textReporter.fileHandle, phReport)
	handleWrite(textReporter.fileHandle, "=== Prompt ========================================\n")
	handleWrite(textReporter.fileHandle, ct.Prompt)
	if len(ct.Memory) > 0 {
		handleWrite(textReporter.fileHandle, "\n=== Memory ========================================\n")
		handleWrite(textReporter.fileHandle, ct.Memory)
	}
	if len(ct.AuthorsNote) > 0 {
		handleWrite(textReporter.fileHandle, "\n=== Author's Note =================================\n")
		handleWrite(textReporter.fileHandle, ct.AuthorsNote)
	}
	return textReporter
}

func (tr *TextReporter) ReportIteration(iteration int) {
	handleWrite(tr.fileHandle,
		fmt.Sprintf("\n\n=== Iteration %-5v ==============================\n", iteration))
	tr.fileHandle.Sync()
}

func (tr *TextReporter) ReportGeneration(resp string) {
	handleWrite(tr.fileHandle, resp)
	tr.fileHandle.Sync()
}

func (tr *TextReporter) close() {
	tr.fileHandle.Close()
}

type Reporters struct {
	JSON *JSONReporter
	Text *TextReporter
	Console *ConsoleReporter
}

func (reporters Reporters) close() {
	reporters.JSON.close()
	reporters.Text.close()
	reporters.Console.close()
}

func (reporters Reporters) ReportIteration(iteration int) {
	reporters.Console.ReportIteration(iteration)
	reporters.Text.ReportIteration(iteration)
}

func (reporters Reporters) ReportGeneration(resp string) {
	reporters.Console.ReportGeneration(resp)
	reporters.Text.ReportGeneration(resp)
}

func (reporters Reporters) SerializeIteration(result *IterationResult) {
	reporters.JSON.SerializeIteration(result)
}

func (ct ContentTest) MakeReporters() Reporters {
	consoleReport := ct.CreateConsoleReporter()
	outputPath := ct.generateOutputPath()
	textReport := ct.CreateTextReporter( outputPath + ".txt")
	jsonReport := CreateJSONReporter( outputPath + ".json")
	return Reporters{
		&jsonReport,
		&textReport,
		&consoleReport,
	}
}