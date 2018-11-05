package tasks

// Task 处理的事务任务
type Task interface {
	GetName() string
	Next() Task
	SetNext(t Task)

	Execute() error
}
