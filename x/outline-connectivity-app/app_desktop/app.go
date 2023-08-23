package main

import (
	"context"
	"encoding/json"
	"outline_sdk_connectivity_test/shared_backend"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) Invoke(input shared_backend.CallInputMessage) shared_backend.CallOutputMessage {
	rawInputMessage, marshallingError := json.Marshal(input)

	if marshallingError != nil {
		return shared_backend.CallOutputMessage{Result: "", Errors: []string{"Invoke: failed to serialize raw invocation input"}}
	}

	var outputMessage shared_backend.CallOutputMessage

	unmarshallingError := json.Unmarshal(shared_backend.SendRawCall(rawInputMessage), &outputMessage)

	if unmarshallingError != nil {
		return shared_backend.CallOutputMessage{Result: "", Errors: []string{"Invoke: failed to parse invocation result"}}
	}

	return outputMessage
}
