package hellarad

import "fmt"

type Subject struct {
	Domain	string	`json:"domain"`
	IP 		string 	`json:"address"`
}

type Result struct {
	Source			string
	AttributeValue	string	
	Message			string
	Success			bool
}

func (r *Result) Prettify() string {
	var message string
	if r.Success == true {
		message = fmt.Sprintf("Details on %s from %s:\n%s", r.AttributeValue, r.Source, r.Message)
	} else {
		message = fmt.Sprintf("Failed to get info from %s! Error: %s", r.Source, r.Message)
	}

    return message
}