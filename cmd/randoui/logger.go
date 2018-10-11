package main

import (
	"fmt"
	"time"

	throttle "github.com/limscoder/grpc-athrottle"
)

type requestLogger struct {
	requestSum int
	acceptSum  int
	rejectSum  int
}

func newRequestLogger() *requestLogger {
	r := &requestLogger{}
	ticker := time.NewTicker(2 * time.Second)
	go func() {
		for t := range ticker.C {
			fmt.Println("requests", t, r.requestSum, r.acceptSum, r.rejectSum)
		}
	}()
	return r
}

func (r *requestLogger) log(event throttle.CounterEvent) {
	if event == throttle.RequestEvent {
		r.requestSum++
	} else if event == throttle.AcceptEvent {
		r.acceptSum++
	} else if event == throttle.RejectEvent {
		r.rejectSum++
	}
}
