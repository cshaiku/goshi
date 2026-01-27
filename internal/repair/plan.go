package repair

type Action struct {
	Code        string
	Description string
	Command     []string
}

type Plan struct {
	Actions []Action
}
