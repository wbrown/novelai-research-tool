package nrt

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func handleWrite(f *os.File, s string) {
	_, err := f.WriteString(s)
	if err != nil {
		log.Fatal(err)
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
	paramReport, _ := json.MarshalIndent(ct.Parameters, "             ", " ")
	fmt.Printf("%v %v\n", ur.blue("Parameters:"), string(paramReport))
	return ur
}

func (cr *ConsoleReporter) ReportIteration(iteration int) {
	fmt.Printf("%v %v / %v\n", cr.blue("Iteration:"),
		iteration+1, cr.ct.Iterations)
	fmt.Printf("%v%v\n", cr.blue("<="),
		strings.Replace(cr.ct.Prompt, "\n", cr.blueNewline, -1))
}

func (cr *ConsoleReporter) ReportGeneration(resp string) {
	fmt.Printf("%v%v\n", cr.green("=>"),
		strings.Replace(resp, "\n", cr.greenNewline, -1))
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
		log.Fatal(err)
	}
	reportWriter.fileHandle, err = os.OpenFile(path,
		os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	reportWriter.iteration = 0
	handleWrite(reportWriter.fileHandle, "[")
	return reportWriter
}

func (reportWriter JSONReporter) write(result *IterationResult) {
	if reportWriter.iteration != 0 {
		handleWrite(reportWriter.fileHandle, ",\n")
	}
	reportWriter.iteration += 1
	serialized, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	handleWrite(reportWriter.fileHandle, string(serialized))
	reportWriter.fileHandle.Sync()
}

func (reportWriter JSONReporter) close() {
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
	iteration  int
}

func (ct ContentTest) CreateTextReporter(path string) (textReporter TextReporter) {
	var err error
	dir := filepath.Dir(path)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		log.Fatal(err)
	}
	textReporter.fileHandle, err = os.OpenFile(path,
		os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	paramsReportBytes, err := json.MarshalIndent(ct.Parameters, "", "")
	if err != nil {
		log.Fatal(err)
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
	handleWrite(textReporter.fileHandle, "=== Prompt ========================================\n")
	handleWrite(textReporter.fileHandle, ct.Prompt)
	return textReporter
}

func (tr *TextReporter) write(resp string) {
	tr.iteration += 1
	handleWrite(tr.fileHandle,
		fmt.Sprintf("\n\n=== Iteration %-5v ==============================\n", tr.iteration))
	handleWrite(tr.fileHandle, resp)
	tr.fileHandle.Sync()
}

func (tr *TextReporter) close() {
	tr.fileHandle.Close()
}
