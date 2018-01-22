package main

import (
	"fmt"
	"time"
)
import "sync"

func main() {
	done := make(chan struct{})
	in := gen(done, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15)
	// Distribute the sq work across two goroutines that both read from in.
	c1 := sq(done, in)
	c2 := sq(done, in)
	c3 := sq(done, in)
	// Consume the merged output from c1 and c2.c1
	ch := merge(done, c1, c2, c3)
	fmt.Println(<-ch)
	close(done)
	time.Sleep(time.Second)
}

func gen(done chan struct{}, nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer func() {
			fmt.Println("gen done")
		}()
		defer close(out)
		for _, n := range nums {
			select {
			case out <- n:
			case <-done:
			}
		}
	}()
	return out
}

func sq(done chan struct{}, in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer func() {
			fmt.Println("sq done")
		}()
		defer close(out)
		for n := range in {
			select {
			case out <- n * n:
			case <-done:
				return
			}
		}
	}()
	return out
}

func merge(done chan struct{}, cs ...<-chan int) <-chan int {
	var wg sync.WaitGroup
	out := make(chan int)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan int) {
		defer func() {
			fmt.Println("output done")
		}()
		for n := range c {
			select {
			case out <- n:
			case <-done:
				return
			}
		}
		wg.Done()
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
