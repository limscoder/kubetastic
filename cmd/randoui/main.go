package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	throttle "github.com/limscoder/grpc-athrottle"

	"github.com/limscoder/kubetastic/pkg/randopb"

	"google.golang.org/grpc"
)

var serviceAddr = flag.String("server", "127.0.0.1:9001", "host:port for server")
var valueHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<title>RANDO</title>
</head>

<body id="home">
	<h1>Your random number is: <span style="color: blue">{{.}}</span></h1>
</body>
</html>
`

func handleRand(writer http.ResponseWriter, req *http.Request, client randopb.RandoClient) {
	seed := time.Now().UnixNano()
	value, err := client.GetRand(context.Background(), &randopb.GetRandRequest{Seed: seed, Max: 100})
	if err != nil {
		http.Error(writer, fmt.Sprintf("server connection failed: %v", err), http.StatusInternalServerError)
		return
	}

	t, err := template.New("value").Parse(valueHTML)
	if err != nil {
		log.Fatalf("template failed: %v", err)
	}
	t.Execute(writer, value.Value)
}

func main() {
	flag.Parse()

	logger := newRequestLogger()
	params := throttle.ThrottleOptions{
		WindowDuration:  3 * time.Minute,
		MinRequestCount: 25,
		MaxRatio:        2.,
		Callback:        logger.log,
	}
	unaryInterceptOpt := grpc.WithUnaryInterceptor(throttle.NewClientUnaryInterceptor(&params))

	randoCon, err := grpc.Dial(*serviceAddr, grpc.WithInsecure(), unaryInterceptOpt)
	if err != nil {
		log.Fatalf("failed to connect to server: %v", err)
	}
	defer randoCon.Close()
	client := randopb.NewRandoClient(randoCon)

	http.HandleFunc("/rand", func(writer http.ResponseWriter, req *http.Request) {
		handleRand(writer, req, client)
	})
	err = http.ListenAndServe(":9002", nil)
	if err != nil {
		log.Fatal(fmt.Sprintf("failed to serve: %v", err))
	}
}
