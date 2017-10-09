package queue

type Queue struct{}

func New(name string) *Queue {
	return &Queue{}
}

func (q *Queue) Start() {
	select {}
}
