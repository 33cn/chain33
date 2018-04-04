package stackimpact

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestMeasureSegment(t *testing.T) {
	agent := NewAgent()

	done1 := make(chan bool)

	var seg1 *Segment
	go func() {
		seg1 = agent.MeasureSegment("seg1")
		defer seg1.Stop()

		time.Sleep(50 * time.Millisecond)

		done1 <- true
	}()

	<-done1

	time.Sleep(10 * time.Millisecond)

	if seg1.Duration < 50 {
		t.Errorf("Duration of seg1 is too low: %v", seg1.Duration)
	}
}

func TestMeasureHandler(t *testing.T) {
	agent := NewAgent()

	// start HTTP server
	go func() {
		http.Handle(agent.MeasureHandler("/test1", http.StripPrefix("/test1", http.FileServer(http.Dir("/tmp")))))

		if err := http.ListenAndServe(":5010", nil); err != nil {
			t.Error(err)
			return
		}
	}()

	waitForServer("http://localhost:5010/test1")

	res, err := http.Get("http://localhost:5010/test1")
	if err != nil {
		t.Error(err)
		return
	} else if res.StatusCode != 200 {
		io.Copy(os.Stdout, res.Body)
		t.Error(err)
		return
	} else {
		defer res.Body.Close()
	}
}

func TestMeasureHandlerFunc(t *testing.T) {
	agent := NewAgent()

	// start HTTP server
	go func() {
		http.HandleFunc(agent.MeasureHandlerFunc("/test2", func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			fmt.Fprintf(w, "OK")
		}))

		if err := http.ListenAndServe(":5011", nil); err != nil {
			t.Error(err)
			return
		}
	}()

	waitForServer("http://localhost:5011/test2")

	res, err := http.Get("http://localhost:5011/test2")
	if err != nil {
		t.Error(err)
		return
	} else if res.StatusCode != 200 {
		t.Error(err)
		return
	} else {
		defer res.Body.Close()
	}
}

func TestRecoverPanic(t *testing.T) {
	agent := NewAgent()

	done := make(chan bool)

	go func() {
		defer func() {
			if err := recover(); err != nil {
				t.Error("panic1 unrecovered")
			}
		}()
		defer agent.RecordAndRecoverPanic()
		defer func() {
			done <- true
		}()

		panic("panic1")
	}()

	<-done
}

func waitForServer(url string) {
	for {
		if _, err := http.Get(url); err == nil {
			time.Sleep(100 * time.Millisecond)
			break
		}
	}
}

func BenchmarkMeasureSegment(b *testing.B) {
	agent := NewAgent()
	agent.Start(Options{
		AgentKey: "key1",
		AppName:  "app1",
	})

	for i := 0; i < b.N; i++ {
		s := agent.MeasureSegment("seg1")
		s.Stop()
	}

	// go test -v -run=^$ -bench=BenchmarkMeasureSegment -cpuprofile=cpu.out
	// go tool pprof internal.test cpu.out
}

func BenchmarkRecordError(b *testing.B) {
	agent := NewAgent()
	agent.Start(Options{
		AgentKey: "key1",
		AppName:  "app1",
	})

	err := errors.New("error1")

	for i := 0; i < b.N; i++ {
		agent.RecordError(err)
	}

	// go test -v -run=^$ -bench=BenchmarkRecordError -cpuprofile=cpu.out
	// go tool pprof internal.test cpu.out
}
