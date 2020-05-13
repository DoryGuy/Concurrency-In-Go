package utils

import (
	"fmt"
	"math/rand"
	"runtime"
	"testing"
	"time"
)

func TestOrChannel(t *testing.T) {
	now := time.Now()
	defer func() {
		fmt.Println("Execution Time: ", time.Since(now))
	}()
	defer time.Sleep(time.Second) // give it time to print the Execution time.

	or := OrChannel

	sig := func(after time.Duration) <- chan interface{}{
		c := make(chan interface{})
		go func() {
			defer close(c)
			time.Sleep(after)
		}()
		return c
	}

	start := time.Now()
	// The first channel to finish will cause "or" to return that result.
	<-or(
		sig(2*time.Hour),
		sig(5*time.Minute),
		sig(1*time.Second),
		sig(1*time.Hour),
		sig(1*time.Minute))
	finishDuration := time.Since(start)
	fmt.Printf("done after %v", finishDuration)
	if !(finishDuration < (1*time.Minute)) || (finishDuration < (1*time.Second)) {
		t.Fatalf("Should have exited in less than a minute and over 1 second")
	}
}

func TestTakeAndRepeatFnChannel(t *testing.T) {
	now := time.Now()
	defer func() {
		fmt.Println("Execution Time: ", time.Since(now))
	}()
	defer time.Sleep(time.Second) // give it time to print the Execution time.

	take := TakeChannel
	repeatFn:= RepeatFnChannel

	done := make(chan interface{})
	defer close(done)

	rand:= func() interface{} {
		return rand.Int()
	}

	for num := range take(done, repeatFn(done, rand), 10) {
		fmt.Println(num)
	}
}

func TestTeeChannel(t *testing.T) {
	now := time.Now()
	defer func() {
		fmt.Println("Execution Time: ", time.Since(now))
	}()
	defer time.Sleep(time.Second)  // give it time to print the Execution time.

	tee := TeeChannel
	take := TakeChannel
	repeat := RepeatValueChannel

	done := make(chan interface{})
	defer close(done)

	out1, out2 := tee(done, take(done, repeat(done, 1, 2), 4))

	for val1 := range out1 {
		val2 := <-out2
		fmt.Printf("Out1: %v, out2: %v\n", val1, val2)
		if val1 != val2 {
			t.Fatalf("Val1: %v != Val2: %v", val1, val2)
		}
	}
}

func TestBridgeChannels(t *testing.T) {
	now := time.Now()
	defer func() {
		fmt.Println("Execution Time: ", time.Since(now))
	}()
	defer time.Sleep(time.Second)  // give it time to print the Execution time.

	bridge := BridgeChannel
	toInt := ToIntChannel

	genVals := func() <- chan <- chan interface{} {
		chanStream := make(chan (<-chan interface{}))
		go func() {
			defer close(chanStream)
			for i := 0; i < 10; i++ {
				stream := make(chan interface{}, 1)
				stream <- i
				close(stream)
				chanStream <- stream
			}
		}()
		return chanStream
	}

	done := make(chan interface{})
	defer close(done)

	var result []int
	expectedResult := []int{ 0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	for v := range toInt(done, bridge(done, genVals())) {
		fmt.Printf("%d ", v)
		result = append(result, v)
	}
	if !IntArrayEquals(result, expectedResult) {
		t.Fatalf("expected %v, \n got %v", expectedResult, result)
	}
	fmt.Printf("Done!")
}

func IntArrayEquals(a []int, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func TestFanIn(t *testing.T) {
	now := time.Now()
	defer func() {
		fmt.Println("Execution Time: ", time.Since(now))
	}()
	defer time.Sleep(time.Second)  // give it time to print the Execution time.

	// primeFinder is from the book, as an example of using fan out/fan in
	// not actually a great algorithm to determine prime numbers.
	// given a number, find the first prime smaller than that number.
	primeFinder := func(done <-chan interface{}, intStream <-chan int) <-chan interface{} {
		primeStream := make(chan interface{})
		go func() {
			defer close(primeStream)
			for integer := range intStream {
				prime := true
				for divisor := (integer + 1)/2; divisor > 1; divisor-- {
					if integer%divisor == 0 {
						prime = false
						break
					}
				}

				if prime {
					select {
					case <-done:
						return
					case primeStream <- integer:
					}
				}
			}
		}()
		return primeStream
	}

	fanIn := FanInChannel
	toInt := ToIntChannel
	repeatFn := RepeatFnChannel
	take := TakeChannel

	done := make(chan interface{})
	defer close(done)

	start := time.Now()

	rand := func() interface{} { return rand.Intn(50000000) }
	randIntStream := toInt(done, repeatFn(done, rand))

	numFinders := 1 + runtime.NumCPU()
	fmt.Printf("Spinning up %d prime finders.\n", numFinders)
	finders := make([]<-chan interface{}, numFinders)
	fmt.Println("Primes:")
	for i := 0; i < numFinders; i++ {
		finders[i] = primeFinder(done, randIntStream)
	}

	for prime := range take(done, fanIn(done, finders...), 10) {
		fmt.Printf("\t%d\n", prime)
	}

	fmt.Printf("Search took: %v", time.Since(start))
}

func TestSliceToChannel(t *testing.T) {
	now := time.Now()
	defer func() {
		fmt.Println("Execution Time: ", time.Since(now))
	}()
	defer time.Sleep(time.Second)  // give it time to print the Execution time.

	generator := GeneratorToChannel
	toFloat64 := ToFloat64Channel

	done := make(chan interface{})
	defer close(done)

	dataChannel := generator(done, 0.1, 0.2, 0.3)
	for val := range toFloat64(done, dataChannel) {
		fmt.Printf("%f\n", val)
	}
}