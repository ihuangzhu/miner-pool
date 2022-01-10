package core

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

type ProxyServer struct {
	srv *http.Server

	wg sync.WaitGroup
}

func StartProxyServer() *ProxyServer {
	r := &ProxyServer{
		srv: &http.Server{Addr: ":8545"},
	}
	r.wg.Add(1)

	go r.listen()
	return r
}

func (p *ProxyServer) listen() {
	defer func() {
		log.Print("ProxyServer is exiting")
		p.wg.Done()
	}()

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

		log.Printf("Received data: %v", string(data))
		resp, err  := p.sendHttpRequest(string(data))
		log.Printf("Response data: %v", string(resp))

		w.Write(resp)
	})

	if err := p.srv.ListenAndServe(); err != http.ErrServerClosed {
		// unexpected error. port in use?
		log.Fatalf("ListenAndServe(): %v", err)
	}
}

func (p *ProxyServer) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := p.srv.Shutdown(ctx); err != nil {
		log.Panicf("Shutdown(): %v", err) // failure/timeout shutting down the svr gracefully
	}

	// wait for goroutine started in startHttpServer() to stop
	p.wg.Wait()
}

// sendHttpRequest 发送请求
func (p *ProxyServer) sendHttpRequest(param string) ([]byte, error) {
	//resp, err := http.Post("http://192.168.198.21:8545", "application/json", bytes.NewBuffer([]byte(param)))
	resp, err := http.Post("http://192.168.198.21:8545", "application/json", bytes.NewBuffer([]byte(param)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}
