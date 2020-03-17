package concurrent

import (
	"log"
	"sync"
)

// Req - req payload for a parallelization task
type Req struct {
	ItemNo int         `json:"item_no"`
	Data   interface{} `json:"data"`
	Config interface{} `json:"config"`
	IsDone bool        `json:"is_done"`
}

// Resp - Resp from a worker
type Resp struct {
	ItemNo  int         `json:"item_no"`
	ReqData interface{} `json:"req_data"`
	Data    interface{} `json:"data"`
	Err     error       `json:"error"`
	IsDone  bool        `json:"is_done"`
}

type ProcessFunc func(msg *Req) (resp *Resp)
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
		log.Printf("CONCURRENT_WORKER_BEGIN: %d, %s\n", i, endMsg)
	}
	c.wg.Done()
}

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

func (c *Concurrent) DrainReqs() {
	go func() {
		for i := 0; i < c.numWorkers; i++ {
			c.inputCh <- &Req{IsDone: true}
		}
	}()
	c.wg.Wait()
	c.outputCh <- &Resp{IsDone: true}
}

func (c *Concurrent) IsErr() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.err != nil
}

func (c *Concurrent) GetErr() error {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.err
}

func (c *Concurrent) setErr(err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.err = err
}
