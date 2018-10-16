package tasks

type Task interface {
	Next() Task
	SetNext(t Task)

	Execute() error
}
