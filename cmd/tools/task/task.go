package task

// Task 处理的事务任务
type Task interface {
	Next() Task
	SetNext(t Task)

	Execute() error
}
