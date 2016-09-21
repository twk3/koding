package awsprovider

import "koding/kites/kloud/stackplan"

var p = &stackplan.Provider{
	Name:       "aws",
	Provider:   "instance",
	NewMachine: newMachine,
	NewStack:   newStack,
	Schema: &stackplan.ProviderSchema{
		NewCredential: func() interface{} { return &Cred{} },
		NewBootstrap:  func() interface{} { return &Bootstrap{} },
		NewMetadata:   func() interface{} { return &Meta{} },
	},
}

func init() {
	stackplan.Register(p)
}

func newMachine(bs *stackplan.BaseMachine) (stackplan.Machine, error) {
	return nil, nil
}

func newStack(bs *stackplan.BaseStack) (stackplan.Stack, error) {
	return nil, nil
}
