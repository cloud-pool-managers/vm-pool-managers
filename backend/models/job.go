package models

type JobType int

const (
	CreateVM JobType = iota
	DeleteVM
	AttribVM
	CreateVolumeAndAttach
)

type Job struct {
	Type JobType
	Data map[string]string
}
