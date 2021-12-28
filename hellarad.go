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

func (r *Result) stringify() string {
	message := fmt.Sprintf("Details on %s from %s:\n", subject.AttributeValue, subject.Source)

    return message
}