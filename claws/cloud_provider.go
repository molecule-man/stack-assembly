package claws

import (
	"errors"
)

// CloudProvider wraps the cloud provider functions
type CloudProvider interface {
	ValidateTemplate(tplBody string) ([]string, error)
	CreateChangeSet(stackName string, tplBody string, params map[string]string) (string, error)
	WaitChangeSetCreated(ID string) error
	ChangeSetChanges(ID string) ([]Change, error)
	ExecuteChangeSet(ID string) error
	WaitStack(stackName string) error
	StackEvents(stackName string) ([]StackEvent, error)
	StackOutputs(stackName string) ([]StackOutput, error)
	BlockResource(stackName string, resource string) error
	UnblockResource(stackName string, resource string) error
}

// Change is a change that is applied to the stack
type Change struct {
	Action            string
	ResourceType      string
	LogicalResourceID string
	ReplacementNeeded bool
}

// StackEvent is a stack event
type StackEvent struct {
	ID                string
	ResourceType      string
	Status            string
	LogicalResourceID string
	StatusReason      string
}

// StackOutput contains info about stack output variables
type StackOutput struct {
	Key         string
	Value       string
	Description string
	ExportName  string
}

//ErrNoChange is error that indicate that there are no changes to apply
var ErrNoChange = errors.New("No changes")
