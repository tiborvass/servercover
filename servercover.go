package servercover

import (
	"flag"
	"net/rpc"
	"os"
	"sync"
	"testing"
	"time"
	"unsafe"
)

var coverAddr = flag.String("cover.addr", "", "Address to servercover")

//go:linkname writeProfiles testing.(*M).writeProfiles
func writeProfiles(m *testing.M)

//go:linkname after testing.(*M).after
func after(m *testing.M)

//go:linkname testingCover *testing.Cover
var testingCover *testing.Cover

// Copied from testing.M
// TODO: add tests to ensure struct is in sync with testing package.
type _M struct {
	deps       interface{}
	tests      []testing.InternalTest
	benchmarks []testing.InternalBenchmark
	examples   []testing.InternalExample

	timer     *time.Timer
	afterOnce sync.Once

	numRun int
}

var conn struct {
	client     *rpc.Client
	coverMutex sync.Mutex
}

type resetCoverRequest bool

const (
	resetCoverStats resetCoverRequest = true
	keepCoverStats                    = false
)

func terminate() {
	if conn.client != nil {
		updateCover(resetCoverStats)
		conn.client.Close()
		//} else {
		//*testingCover = testing.Cover{}
		//testingCover.Counters = nil
		//testingCover.Blocks = nil
	}
}

func customAfter(m *testing.M) {
	var newAfterOnce sync.Once
	newAfterOnce.Do(func() {
		terminate()
		writeProfiles(m)
	})
}

func updateCover(shouldResetCover resetCoverRequest) {
	var updatedCover testing.Cover
	if err := conn.client.Call("ServerCover.GetCover", shouldResetCover, &updatedCover); err != nil {
		panic(err)
	}
	conn.coverMutex.Lock()
	defer conn.coverMutex.Unlock()
	testing.RegisterCover(updatedCover)
}

// TestMain is what needs to be called from the test package's TestMain function.
//
//  func TestMain(m *testing.M) {
//    servercover.TestMain(m)
//  }
func TestMain(m *testing.M) {
	flag.Parse()
	if *coverAddr == "" {
		panic("-cover.addr is needed")
	}
	network, addr := "unix", *coverAddr

	// disable m.after()
	(*_M)(unsafe.Pointer(m)).afterOnce.Do(func() {})

	var emptyCover = testing.Cover{Mode: "set"}
	testing.RegisterCover(emptyCover)

	var exitCode int

	// closure to ensure deferred calls happen before os.Exit
	func() {
		// defer modified m.after()
		defer customAfter(m)

		exitChan := make(chan int)
		go func() {
			// run tests
			exitChan <- m.Run()
		}()

		var err error
		waitTime := time.Second
		maxAttempts := 3
		for i := 0; i < maxAttempts; i++ {
			if err != nil {
				time.Sleep(waitTime)
				waitTime = waitTime << 1
			}
			conn.client, err = rpc.Dial(network, addr)
		}
		if err != nil {
			panic(err)
		}
		updateCover(keepCoverStats)
		exitCode = <-exitChan
	}()
	os.Exit(exitCode)
}

func Coverage() float64 {
	updateCover(keepCoverStats)
	return testing.Coverage()
}
