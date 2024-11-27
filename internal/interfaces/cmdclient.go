package interfaces

type CommandClient interface {
	RunCommand(command string, arguments []string) (string, error)
	Username() string
	Password() string
}
