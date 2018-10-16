package tasks

type TaskBase struct {
	NextTask Task
}

func (this *TaskBase) Execute() error {
	return nil
}

func (this *TaskBase) SetNext(t Task) {
	this.NextTask = t
}

func (this *TaskBase) Next() Task {
	return this.NextTask
}
