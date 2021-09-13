package trace

import "testing"
import(
trdht "github.com/libp2p/dht-tracer1/tracedht"
)

func TestNew(t *testing.T) {
	var waitChan =make(chan struct{})
	 New(nil)

	<-waitChan

}

func TestTracer(t *testing.T){
	var opts trdht.Opts


}