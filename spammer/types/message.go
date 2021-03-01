package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Processor defines information relevant to msg processing
type Processor struct {
	Order              int  `toml:"order" json:"order"`
	Count              int  `toml:"count" json:"count"`
	FromPrimaryAccount bool `toml:"from_primary_account" json:"from_primary_account"`
	AfterWaitSeconds   int  `toml:"after_wait_seconds" json:"after_wait_seconds"`
}

// Validate validates Processor fields
func (p Processor) Validate() error {
	if p.Order < 0 {
		return fmt.Errorf("order field cannot be negative")
	}
	if p.Count < 0 {
		return fmt.Errorf("count field cannot be negative")
	}
	if p.AfterWaitSeconds < 0 {
		return fmt.Errorf("after wait seconds field cannot be negative")
	}
	return nil
}

// Message defines a sdk.Msg and processing information
type Message struct {
	Msg       sdk.Msg   `json:"msg" yaml:"msg"`
	Processor Processor `json:"processor" yaml:"processor"`
}

// Validate validates Message fields
func (m Message) Validate() error {
	if m.Msg.Type() == "" {
		return fmt.Errorf("sdk.Msg must define a type")
	}
	err := m.Processor.Validate()
	if err != nil {
		return err
	}
	return nil
}

// Messages is a slice of Message
type Messages []Message

// Validate validates a slice of Messages
func (ms Messages) Validate() error {
	if len(ms) == 0 {
		return fmt.Errorf("must define at least one message")
	}
	for _, m := range ms {
		err := m.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}
