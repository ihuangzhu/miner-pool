package core

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

type Receiver struct {
	proxy *Proxy

	srv *http.Server

	wg sync.WaitGroup
}

func StartReceiver(proxy *Proxy) *Receiver {
	r := &Receiver{
		proxy: proxy,
		srv:   &http.Server{Addr: *proxy.svr.cfg.Proxy.Daemon.NotifyWorkUrl},
	}
	r.wg.Add(1)

	go r.listen()
	return r
}

func (r *Receiver) listen() {
	defer r.wg.Done()

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "POST" {
			log.Print("Invalid HTTP method")
			return
		}

		// 读取数据
		data, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Print("Unable to read work notification")
			return
		}

		// 解析任务数据
		var notification []string
		err = json.Unmarshal(data, &notification)
		if err != nil {
			log.Print("Unable to parse work notification")
			return
		}

		r.proxy.sender.workCh <- notification
		log.Printf("Received notification: %v", notification)
	})

	if err := r.srv.ListenAndServe(); err != http.ErrServerClosed {
		// unexpected error. port in use?
		log.Fatalf("ListenAndServe(): %v", err)
	}
}

func (r *Receiver) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := r.srv.Shutdown(ctx); err != nil {
		log.Panicf("Shutdown(): %v", err) // failure/timeout shutting down the svr gracefully
	}

	// wait for goroutine started in startHttpServer() to stop
	r.wg.Wait()
}
