package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/tidwall/pretty"
)

var (
	Sources = []string{"OpsGenie", "Splunk", "Sumo Logic"}
	Outputs = []string{"OpsGenie", "Jira"}
)

type sqFunc struct {
	Name            string
	Type            string
	Supports        string
	KeyLocation     string
	FunctionName    string
	FunctionNameArn string
}

type stateTemplate struct {
	MultiTasks string
	IPv4Tasks  string
}

func main() {
	//----- Choose the alert source
	alertSource := promptOptions("Alert source", "Select alert source", Sources)

	topicPolicy, err := ioutil.ReadFile("../../template/opsgenie_sns.yaml")
	if err != nil {
		log.Fatal(err)
	}

	if alertSource == "OpsGenie" {
		topicPolicy = append([]byte("Resources:\n"), topicPolicy...)
		replaceInFile("../../template.yaml", []byte("Resources:"), topicPolicy, true)
	} else {
		topicPolicy = append(topicPolicy, []byte("\n")...)
		replaceInFile("../../template.yaml", topicPolicy, []byte(""), true)
	}

	//----- Choose the output provider
	output := promptOptions("Output", "Select output destination", Outputs)

	if output == "OpsGenie" {
		replaceInFile("../../template.yaml", []byte("output/jira"), []byte("output/opsgenie"), false)
		replaceInFile("../../template.yaml", []byte("Handler: jira"), []byte("Handler: opsgenie"), false)
		replaceInFile("../../template.yaml", []byte("          PROJECT: .*\n"), []byte("          PROJECT: unused by opsgenie\n"), false)
		replaceInFile("../../template.yaml", []byte("          BASE_URL: .*\n"), []byte("          BASE_URL: unused by opsgenie\n"), false)
	} else {
		replaceInFile("../../template.yaml", []byte("output/opsgenie"), []byte("output/jira"), false)
		replaceInFile("../../template.yaml", []byte("Handler: opsgenie"), []byte("Handler: jira"), false)

		project := fmt.Sprintf("          PROJECT: %s", promptInput("Configure Jira", "Enter your JIRA project name.", "[A-Z]+"))
		replaceInFile("../../template.yaml", []byte("          PROJECT: .*\n"), []byte(project), false)

		baseUrl := fmt.Sprintf("          BASE_URL: %s", promptInput("Configure Jira", "Enter your JIRA base URL.", "https://[a-z\\-]+.atlassian.net"))
		replaceInFile("../../template.yaml", []byte("          BASE_URL: .*\n"), []byte(baseUrl), false)
	}

	//----- Choose enrichment functions
	var useFunctions []sqFunc

	files, err := ioutil.ReadDir("../../function")
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if file.IsDir() {
			fileName := fmt.Sprintf("../../function/%s/main.go", file.Name())
			provider, err := getProviderInfo(fileName)
			if err != nil {
				log.Fatal(err)
			}
			cls(provider.Name)

			fmt.Printf("Found %s provider '%s'\n\n", provider.Type, provider.Name)
			fmt.Printf("Supports: %s\n", provider.Supports)
			fmt.Printf("Lambda Function Name: %s\n", provider.FunctionName)

			if provider.KeyLocation == "" {
				fmt.Printf("API Key not required.\n\n")
			} else {
				fmt.Printf("API Key Secrets Manager Location: %s\n\n", provider.KeyLocation)
			}

			if promptYesNo("Use this provider?") == true {
				useFunctions = append(useFunctions, provider)
			}
		}
	}
	if len(useFunctions) == 0 {
		log.Fatal("You must select at least one provider! Abort.")
	}

	cls("Confirm")
	fmt.Println("You have selected:")
	for _, selected := range useFunctions {
		fmt.Printf("- %s (%s)\n", selected.Name, selected.Type)
	}
	if promptYesNo("\nIs this correct?") == false {
		fmt.Println("Cancelled.")
		return
	}

	cls("Build")
	fmt.Println("-+. building state machine ASL definition ..")

	var state stateTemplate

	task, terr := template.ParseFiles("../../statemachine/templates/task.tmp")
	pass, perr := template.ParseFiles("../../statemachine/templates/pass.tmp")
	templ, tmperr := template.ParseFiles("../../statemachine/templates/template.tmp")

	if terr != nil || perr != nil || tmperr != nil {
		panic("Failed to load templates! Abort.")
	}

	for _, fn := range useFunctions {
		buf := new(bytes.Buffer)
		err = task.Execute(buf, fn)
		if err != nil {
			log.Fatal(err)
		}

		if fn.Type == "multipurpose" {
			state.MultiTasks = state.MultiTasks + buf.String()
		} else if fn.Type == "ipv4" {
			state.IPv4Tasks = state.IPv4Tasks + buf.String()
		}
	}

	if state.MultiTasks == "" {
		buf := new(bytes.Buffer)
		err = pass.Execute(buf, &sqFunc{
			Type: "multipurpose",
		})
		if err != nil {
			log.Fatal(err)
		}
		state.MultiTasks = buf.String()
	}

	if state.IPv4Tasks == "" {
		buf := new(bytes.Buffer)
		err = pass.Execute(buf, &sqFunc{
			Type: "ipv4",
		})
		if err != nil {
			log.Fatal(err)
		}
		state.IPv4Tasks = buf.String()
	}

	state.MultiTasks = strings.TrimRight(state.MultiTasks, ",")
	state.IPv4Tasks = strings.TrimRight(state.IPv4Tasks, ",")

	buf := new(bytes.Buffer)
	err = templ.Execute(buf, state)
	prettyJson := pretty.Pretty(buf.Bytes())

	fmt.Println("-+. done.")

	cls("Save")

	outFile := "../../statemachine/enrich.asl.json"
	if promptYesNo("Overwrite main state machine definition (enrich.asl.json)?") == false {
		now := time.Now()
		outFile = fmt.Sprintf("../../statemachine/bootstrap-%s.asl.json", now.Format("20060102150405"))
	}

	fmt.Printf("-+. saving to '%s' ..\n", outFile)
	_ = os.WriteFile(outFile, prettyJson, 0644)

	fmt.Println("-+. ok done, bootstrap run complete. woot!!")
}

func replaceInFile(file string, replace []byte, with []byte, backup bool) {
	re := regexp.MustCompile(string(replace))

	orig, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}

	if strings.Contains(string(orig), string(with)) && string(with) != "" {
		fmt.Print("Replace not necessary. Text already present.")
		return
	}
	output := re.ReplaceAll(orig, with)
	output = bytes.Replace(orig, replace, with, -1)

	if backup {
		if err = ioutil.WriteFile(file+".backup", orig, 0644); err != nil {
			log.Fatal(err)
		}
	}

	if err = ioutil.WriteFile(file, output, 0644); err != nil {
		log.Fatal(err)
	}
}

func cls(header string) {
	fmt.Print("\033[H\033[2J")
	fmt.Printf(logo)
	head := fmt.Sprintf(".:+[ %s ]:~.--", header)

	pad := strings.Repeat("-", 57-len(head))
	fmt.Printf("%s%s --- --  -\n\n", head, pad)
	fmt.Printf(colReset)
}

func grep(filename string, str string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Splits on newlines by default.
	scanner := bufio.NewScanner(f)

	// https://golang.org/pkg/bufio/#Scanner.Scan
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), str) {
			return scanner.Text(), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", nil
}

func getProviderInfo(fn string) (sqFunc, error) {
	var ptype string

	providerLine, err := grep(fn, "provider")
	supportsLine, err := grep(fn, "supports")
	secretLocLine, err := grep(fn, "secretLocation")

	if err != nil {
		return sqFunc{}, err
	}

	name := strings.Split(providerLine, "\"")[1]
	supports := strings.Split(supportsLine, "\"")[1]
	fnName := fmt.Sprintf("%sFunction", strings.ReplaceAll(name, " ", ""))

	if secretLocLine != "" {
		secretLocLine = strings.Split(secretLocLine, "\"")[1]
	}

	if len(strings.Split(supports, ",")) == 1 {
		ptype = supports
	} else {
		ptype = "multipurpose"
	}

	return sqFunc{
		Name:            name,
		Type:            ptype,
		Supports:        supports,
		KeyLocation:     secretLocLine,
		FunctionName:    fnName,
		FunctionNameArn: fmt.Sprintf("${%sArn}", fnName),
	}, nil
}

func promptYesNo(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [Y/n]: ", prompt)
	resp, _ := reader.ReadString('\n')
	answer := strings.ToLower(strings.TrimSuffix(resp, "\n"))

	if answer == "n" || strings.Contains(answer, "no") {
		return false
	}

	return true
}

func promptOptions(heading string, prompt string, options []string) string {
	var answer int
	reader := bufio.NewReader(os.Stdin)

	for !(answer > 0 && answer < len(options)+1) {
		cls(heading)
		index := 1
		for _, option := range options {
			fmt.Printf("%x. %s\n", index, option)
			index += 1
		}
		fmt.Println("")

		fmt.Printf("\r%s [1 - %x]: ", prompt, len(options))
		resp, _ := reader.ReadString('\n')
		answer, _ = strconv.Atoi(strings.TrimSuffix(resp, "\n"))
	}

	return options[answer-1]
}

func promptInput(heading string, prompt string, validRegex string) string {
	var answer string
	match := false

	reader := bufio.NewReader(os.Stdin)

	for !match {
		cls(heading)
		fmt.Printf("\r%s Must match regex '%s'!\n: ", prompt, validRegex)
		answer, _ = reader.ReadString('\n')
		match, _ = regexp.MatchString(validRegex, answer)
	}

	return answer
}

var logo = `
,adPPYba,  ,adPPYb,d8 88       88 8b       d8 8b,dPPYba,  ,adPPYba,      
I8[    "" a8"     Y88 88       88  8b     d8' 88P'   "Y8 a8P_____88      
 "Y8ba,   8b       88 88       88   8b   d8'  88         8PP"""""""      
aa    ]8I "8a    ,d88 '8a,   ,a88    8b,d8'   88         '8b,   ,aa
 'YbbdP"    "YbbdP 88    YbbdP'Y8     Y88'    88           iYbbd8' 
                   88                 d8'                                
                   88                 d8'                   
`

const (
	colReset  = "\033[0m"
	colRed    = "\033[31m"
	colGreen  = "\033[32m"
	colYellow = "\033[33m"
	colBlue   = "\033[34m"
	colPurple = "\033[35m"
	colCyan   = "\033[36m"
	colWhite  = "\033[37m"
)
