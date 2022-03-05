package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/tidwall/pretty"
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
		fmt.Println("here-multi")
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
		fmt.Println("here-ipv4")
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
