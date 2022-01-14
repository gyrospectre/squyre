package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type StateMachine struct {
	Comment        string  `json:"Comment"`
	StartAt        string  `json:"StartAt"`
	TimeoutSeconds int     `json:"TimeoutSeconds"`
	Version        string  `json:"Version"`
	States         []State `json:"States"`
}

type State struct {
}

func main() {
	jsonFile, err := os.Open("../../statemachine/enrichIP.asl.json")
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Successfully Opened users.json")
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()
	var sm StateMachine

	byteValue, _ := ioutil.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &sm)
	fmt.Println(sm)
}
