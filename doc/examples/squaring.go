package main

/*
pipe line: 模型

1. 很多时候，要完成一个任务要经历很多步骤，每个步骤的输出是另外一个步骤的输出
2. 如果说有的消息传递都是同一个，输入输出的内容用同一个类型的chan,那么pipe就可以互相嵌套
3. 这个非常类似linux中的pipe
*/
import "fmt"

func gen(nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		for _, n := range nums {
			out <- n
		}
		close(out)
	}()
	return out
}

func sq(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		for n := range in {
			out <- n * n
		}
		close(out)
	}()
	return out
}

func main() {
	// Set up the pipeline.
	c := gen(2, 3)
	out := sq(c)

	// Consume the output.
	fmt.Println(<-out) // 4
	fmt.Println(<-out) // 9
	_, ok := <-out
	if !ok {
		fmt.Println("channel is closed")
	}
}
