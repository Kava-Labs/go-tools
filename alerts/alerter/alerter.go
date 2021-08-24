package alerter

type Alerter interface {
	SendMessage(destination string, text string) error
	Info(destination string, text string) error
	Warn(destination string, text string) error
	Error(destination string, text string) error
}
