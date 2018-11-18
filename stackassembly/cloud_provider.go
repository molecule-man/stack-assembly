package stackassembly

import (
	"errors"
	"time"
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
	StackResources(stackName string) ([]StackResource, error)
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
	Timestamp         time.Time
}

// StackOutput contains info about stack output variables
type StackOutput struct {
	Key         string
	Value       string
	Description string
	ExportName  string
}

type StackResource struct {
	LogicalID    string
	PhysicalID   string
	Status       string
	StatusReason string
	Type         string
	Timestamp    time.Time
}

//ErrNoChange is error that indicate that there are no changes to apply
var ErrNoChange = errors.New("no changes")
