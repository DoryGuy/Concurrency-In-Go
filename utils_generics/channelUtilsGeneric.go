package utils_generics

import (
	"sync"
)

// Utilities from  "Concurrency In Go"
// Note: All of these utilities are interuptible via a "done" channel.
//       close the done channel and the utility will close the channel
//       it has created.
//
// Also these channels are compositable, see examples in the test code,
// or read the book.

// adapted from https://github.com/kat-co/concurrency-in-go-src


// OrChannel for combining one or more done channels into a single done that closes
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
func RepeatValueChannel(done <- chan interface{}, values ...interface{}) <-chan interface{} {
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
func RepeatFnChannel(done <- chan interface{}, fn func() interface{}) <-chan interface{} {
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

// OrDoneChannel encapsulate checking for done channels,
// It will continue to pass along the values from a channel until the done channel is closed,
// or the channel passed in is closed.  Useful with a raw channel
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
// For instance, a series of workers reading from a channel generating output that needs
// to be passed along to the next channel.
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
// similar to the UNIX tee command.
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

// GeneratorToChannel, given a slice, convert it to a channel
// This version uses the generic interface{} which has a minor cost of conversion.
// pp.104
func GeneratorToChannel(done <-chan interface{}, slice ...interface{}) <- chan interface{}{
	interfaceChannel := make(chan interface{}, len(slice))
	go func() {
		defer close(interfaceChannel)
		for _, i := range slice {
			select {
			case <-done:
				return
			case interfaceChannel <- i:
			}
		}
	}()
	return interfaceChannel
}

// I keep thinking that I should be able to pass in an []string to a fn which is declared
// to be a []interface but it doesn't work. Probably because the data structs are of different sizes.
// and it's expensive to convert an array.
func GeneratorFromStringArrayToChannel(done <-chan interface{}, slice []string) <- chan interface{}{
	interfaceChannel := make(chan interface{}, len(slice))
	//interfaceChannel := make(chan interface{}, 1)
	go func() {
		defer close(interfaceChannel)
		for _, i := range slice {
			select {
			case <-done:
				return
			case interfaceChannel <- i:
			}
		}
	}()
	return interfaceChannel
}

// Will limit the number of items passed along in the channel to "limit"
// This is to prevent downstream process from being flooded.
func BufferChannel(done <-chan interface{}, in <- chan interface{}, limit int) <- chan interface{}{
	orDone := OrDoneChannel
	interfaceChannel := make(chan interface{}, limit)

	go func() {
		defer func() {
			// clean up the channels we create.
			close(interfaceChannel)
		}()

		for val := range orDone(done, in) {
			select {
			case <-done:
				return
			case interfaceChannel <- val:
			}
		}
	}()

	return interfaceChannel
}

// ToTChannel Take an interface channel and convert it to a type T channel
func ToTChannel [T any] (done <-chan interface{}, valueStream <-chan interface{}) <-chan T {
	theStream := make(chan T)
	go func() {
		defer close(theStream)
		for v := range valueStream {
			select {
			case <-done:
				return
			case theStream <- v.(T):
			}
		}
	}()
	return theStream
}

