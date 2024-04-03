package alerter

type Alerter interface {
	SendMessage(text string) error
	Info(text string) error
	Warn(text string) error
	Error(text string) error
}
