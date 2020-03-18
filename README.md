# Concurrent - Go Package

- Provides easy abstractions to concurrently work on a list of tasks using a pool of worker goroutines.
- Makes sure all worker goroutines exit once the entire list has been processed and there are no leaked goroutines
- **BailOnError support**: Provides support to read error through `IsErr()` API to stop the source of tasks (like reading from a file or database) upon an error in any one of the workers to implement bailOnError functionality

API Documentation in [GoDoc](https://godoc.org/github.com/kidambisrinivas/concurrent)

## Usage

```go
import (
    concurrentlib "github.com/kidambisrinivas/concurrent"
)

concurrent := concurrentlib.NewConcurrent(50, true, true, processTask,
    func(resp *concurrentlib.Resp) {
        respData, _ := resp.Data.(*MyResult)
        results[respData.Task] = respData.Output
    })
for task := range tasks {
    // Bail on error
    if concurrent.IsErr() {
        break
    }
    concurrent.SendTask(&concurrentlib.Req{Data: task, Config: redisClient})
}
concurrent.DrainReqs()
if concurrent.IsErr() {
    return concurrent.GetErr()
}

// procesTask - Function to process a single task
func procesTask(req *concurrentlib.Req) (resp *concurrentlib.Resp) {
    resp = &concurrentlib.Resp{
		ItemNo:  req.ItemNo,
		ReqData: req.Data,
		Data:    nil,
		Err:     nil,
	}

    task, ok := req.Data.(string)
    if !ok {
		resp.Err = fmt.Errorf("BAD_DATA_ERR: (type %v, data %v)", reflect.TypeOf(req.Data), req.Data)
		return resp
    }

    resp.Data := &MyResult{}
    return resp
}
```

## Development Environment

```bash
Programming Language: Golang 1.13.7
Interface: Library APIs / struct
BuildTool/DepTool: Go Mod
```

## Development

```bash
gvm install go1.13.7 -B
gvm use go1.13.7 && export GOPATH=$HOME/gocode
git clone git@github.com:kidambisrinivas/concurrent.git
cd concurrent

go build
go mod vendor
go install
```

## Testing

Coming soon

## Install

```bash
go get -u github.com/kidambisrinivas/concurrent
```
