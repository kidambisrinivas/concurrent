package concurrent

import (
	"log"
	"sync"
)

// Req - req payload for a single task
type Req struct {
	ItemNo    int         `json:"item_no"`
	Data      interface{} `json:"data"`
	Config    interface{} `json:"config"` // Any config object needed to process a task
	IsDone    bool        `json:"is_done"`
	WorkerNum int         `json:"worker_num"`
}

// Resp - Resp from a worker for a single task
type Resp struct {
	ItemNo  int         `json:"item_no"`
	ReqData interface{} `json:"req_data"`
	Data    interface{} `json:"data"`
	Err     error       `json:"error"`
	IsDone  bool        `json:"is_done"`
}

// ProcessFunc - Process function for a single task
type ProcessFunc func(msg *Req) (resp *Resp)

// HandleOutputFunc - Process output of a single task
type HandleOutputFunc func(resp *Resp)

// Concurrent - Takes a function and runs it concurrently
// in many worker goroutines
type Concurrent struct {
	numWorkers     int
	verboseLogging bool
	bailOnError    bool
	processFn      ProcessFunc
	outputFn       HandleOutputFunc

	err      error
	mutex    *sync.RWMutex
	inputCh  chan *Req
	outputCh chan *Resp
	wg       *sync.WaitGroup
}

// NewConcurrent - Returns a new concurrent runner
func NewConcurrent(numWorkers int,
	verboseLogging bool,
	bailOnError bool,
	fn ProcessFunc,
	handleOutputFn HandleOutputFunc) (c *Concurrent) {
	c = &Concurrent{
		numWorkers:     numWorkers,
		verboseLogging: verboseLogging,
		bailOnError:    bailOnError,
		processFn:      fn,
		outputFn:       handleOutputFn,

		err:      nil,
		mutex:    &sync.RWMutex{},
		inputCh:  make(chan *Req, 0),
		outputCh: make(chan *Resp, 0),
		wg:       &sync.WaitGroup{},
	}
	for i := 0; i < c.numWorkers; i++ {
		c.wg.Add(1)
		go c.worker(i)
	}
	go c.outputListener()
	return c
}

// worker - Worker to perform a certain task
func (c *Concurrent) worker(i int) {
	if c.verboseLogging {
		log.Printf("CONCURRENT_WORKER_BEGIN: %d\n", i)
	}
	endMsg := ""
MAIN_LOOP:
	for {
		select {
		case req := <-c.inputCh:
			if req.IsDone {
				endMsg = "End message sent!"
				break MAIN_LOOP
			}
			req.WorkerNum = i
			resp := c.processFn(req)
			if resp.Err != nil {
				log.Printf("CONCURRENT_WORKER_ERR: (worker %d, %v)\n", i, resp.Err)
				if !c.IsErr() {
					c.setErr(resp.Err)
				}
			}
			c.outputCh <- resp
		}
	}
	if c.verboseLogging {
		log.Printf("CONCURRENT_WORKER_END: Closing %d, %s\n", i, endMsg)
	}
	c.wg.Done()
}

// outputListener - Listen for outputs from all workers and
// calls the ProcessOutput function sent to Concurrent object
func (c *Concurrent) outputListener() {
OUTPUT_LOOP:
	for {
		select {
		case resp := <-c.outputCh:
			if resp.IsDone {
				break OUTPUT_LOOP
			}
			c.outputFn(resp)
		}
	}
	if c.verboseLogging {
		log.Printf("CONCURRENT_OUTPUT_LISTENER_CLOSE\n")
	}
}

// SendTask - Send task to a worker
func (c *Concurrent) SendTask(req *Req) {
	c.inputCh <- req
}

// DrainReqs - Waits for workers to complete in-flight tasks
// and sends terminate/done command to all of them
func (c *Concurrent) DrainReqs() {
	go func() {
		for i := 0; i < c.numWorkers; i++ {
			c.inputCh <- &Req{IsDone: true}
		}
	}()
	c.wg.Wait()
	c.outputCh <- &Resp{IsDone: true}
}

// IsErr - Check if an error occured in processing of any task till now
// CONCURRENCY_SAFE
func (c *Concurrent) IsErr() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.err != nil
}

// GetErr - Gets the error occured in processing of any task till now if any
// CONCURRENCY_SAFE
func (c *Concurrent) GetErr() error {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.err
}

// setErr - Set error of Concurrent object from the error of any task
// CONCURRENCY_SAFE
func (c *Concurrent) setErr(err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.err = err
}
