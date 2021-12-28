package main

import (
	"miner-pool/cmd"
	"os"
)

func main() {
	if err := cmd.Run(os.Args[1:]); err != nil {
		os.Exit(1)
	}

	//interruptChan := make(chan os.Signal, syscall.SIGHUP)
	//
	//s := core.NewSender()
	//s.Attach(&Session{name: "t1"})
	//s.Attach(&Session{name: "t2"})
	//s.Attach(&Session{name: "t3"})
	//
	//listen := ":8107"
	//cfg := &config.Daemon{
	//	NotifyWorkUrl: &listen,
	//}
	//r := core.StartReceiver(cfg, s)
	//defer r.Close()
	//
	//signal.Notify(interruptChan, syscall.SIGINT, syscall.SIGTERM)
	//<-interruptChan
	//fmt.Println("closed")
}
