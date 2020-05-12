package utils

import (
	"sync"
)

// Utilities from  "Concurrency In Go"
// for combining one or more done channels into a single done that closes
// if any of it's component channels close pp. 94-95
//
// Use by creating a variable like this:
//     or := utils.orChannel
//
//     <-or ( doneChannel1, doneChannel2,.... )
func OrChannel(channels ... <-chan interface{}) <-chan interface{} {
	switch len(channels) {
	case 0:
		return nil
	case 1:
		return channels[0]
	}

	orDone := make(chan interface{})
	go func() {
		defer close(orDone)

		switch len(channels) {
		case 2:
			select {
				case <- channels[0]:
				case <- channels[1]:
			}
		default:
			select {
				case <- channels[0]:
				case <- channels[1]:
				case <- channels[2]:
				case <- OrChannel(append(channels[3:], orDone)...):
			}
		}
	}()
	return orDone
}

// RepeatChannel will repeat the values you pass to it infinitely until you tell it to stop.
// pp. 109
func RepeatValue(done <- chan interface{}, values ...interface{}) <-chan interface{} {
	valStream := make(chan interface{})
	go func() {
		defer close(valStream)
		for {
			for _, v := range values {
				select {
				case <-done:
					return
				case valStream <- v:
				}
			}
		}
	}()
	return valStream
}

// RepeatFuncChannel will call the func you pass to it infinitely until you tell it to stop.
// pp. 109
func RepeatFn(done <- chan interface{}, fn func() interface{}) <-chan interface{} {
	valStream := make(chan interface{})
	go func() {
		defer close(valStream)
		for {
			select {
			case <-done:
				return
			case valStream <- fn():
			}
		}
	}()
	return valStream
}

// TakeChannel will only take the first num items from the incoming stream.
// pp. 110
func TakeChannel(done <- chan interface{}, valueStream <-chan interface{}, num int) <-chan interface{} {
	takeStream := make(chan interface{})
	go func() {
		defer close(takeStream)
			for i := 0; i < num; i++ {
				select {
				case <-done:
					return
				case takeStream <- <- valueStream:
				}
			}
	}()
	return takeStream
}

// OrDoneChannel encapsulate checking for done channels
// pp.119-120
func OrDoneChannel(done <-chan interface{}, c <-chan interface{}) <-chan interface{} {
	valStream := make(chan interface{})
	go func() {
		defer close(valStream)
		for {
			select {
			case <-done:
				return
			case v, ok := <-c:
				if ok == false {
					return
				}
				select {
				case valStream <- v:
				case <-done:
				}
			}
		}
    }()
	return valStream
}

// Join multiple streams of data into one single stream
// pp. 117
func FanInChannel(done <-chan interface{}, channels ... <-chan interface{}) <-chan interface{} {
    var wg sync.WaitGroup
    multiplexedStream := make(chan interface{})

    multiplex := func(c <- chan interface{}) {
    	defer wg.Done()
    	for i := range c {
    		select {
    		case <- done:
				return
			case multiplexedStream <- i:
			}
		}
	}

	// Select from all the channels
	wg.Add(len(channels))
    for _,c := range channels {
    	go multiplex(c)
	}

	// Wait for all the reads to complete
	go func() {
		wg.Wait()
		close(multiplexedStream)
	}()
    return multiplexedStream
}

// TeeChannel take the input from the incoming channel and split into two outgoing channels
// pp.120
func TeeChannel(done <-chan interface{}, in <- chan interface{}) (<-chan interface{}, <-chan interface{}) {
	out1 := make(chan interface{})
	out2 := make(chan interface{})
	go func() {
		defer func() {
			close(out1)
			close(out2)
		}()
		orDone := OrDoneChannel
		for val := range orDone(done, in) {
			var out1, out2 = out1, out2 // shadow vars on purpose
			for i := 0; i < 2; i++ {
				select {
				case <-done:
				case out1<-val:
					out1 = nil
				case out2<-val:
					out2 = nil
				}
			}
		}
	}()
    return out1, out2
}

// Bridging multiple channels
// pp.122-123
func BridgeChannel(done <-chan interface{}, chanStream <- chan <- chan interface{}) <-chan interface{} {
	orDone := OrDoneChannel
	valStream := make(chan interface{})
	go func() {
		defer close(valStream)
		for {
			var stream <-chan interface{}
			select {
			case maybeStream, ok := <-chanStream:
				if ok == false {
					return
				}
				stream = maybeStream
			case <-done:
				return
			}

			for val := range orDone(done, stream) {
				select {
				case valStream <- val:
				case <-done:
				}
			}
		}
	}()
	return valStream
}
