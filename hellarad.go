package hellarad

type Subject struct {
	Domain	string	`json:"domain"`
	IP 		string 	`json:"address"`
}

type Result struct {
	Message	string
	Success	bool
}