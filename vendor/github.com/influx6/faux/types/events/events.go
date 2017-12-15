package events

import "github.com/influx6/faux/types/actions"

// FileCreated defines a struct for containing details of file created operation.
type FileCreated struct {
	Action  actions.CreateFile
	Error   error `json:"error"`
	Written int64 `json:"written"`
}

// DataWrittenToFile defines an event for describe a write to file operation.
type DataWrittenToFile struct {
	Action  actions.WriteToFile
	Error   error `json:"error"`
	Written int64 `json:"written"`
}

// DirCreated defines a struct for containing details of dir created operation.
type DirCreated struct {
	Action actions.MkDirectory
	Error  error `json:"error"`
}

// UserCreated defines a struct for defining details of a created user.
type UserCreated struct {
	Action actions.CreateUser
	Error  error `json:"error"`
}
