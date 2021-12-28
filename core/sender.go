package core

import (
	"miner-pool/config"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	// staleThreshold is the maximum depth of the acceptable stale but valid ethash solution.
	staleThreshold = 7
)

type Subscriber interface {
	Notify(work []string)
}

type Sender struct {
	WorkTarget  string
	LastWork    []string
	WorkHistory map[string][]string
	workCh      chan []string

	subs []Subscriber

	quit chan struct{}
	wg   sync.WaitGroup
}

func NewSender(cfg *config.Proxy) *Sender {
	s := &Sender{
		WorkTarget:  *cfg.Target,
		WorkHistory: make(map[string][]string),
		workCh:      make(chan []string, 4),

		quit: make(chan struct{}),
	}

	s.wg.Add(1)
	go s.loop()
	return s
}

func (s *Sender) loop() {
	defer s.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.quit:
			return

		case work := <-s.workCh:
			s.LastWork = work
			s.WorkHistory[work[0]] = work
			notification := s.GetLastWork()
			for _, v := range s.subs {
				go v.Notify(notification)
			}

		case <-ticker.C:
			// Clear stale pending blocks
			if s.LastWork != nil {
				for hash, work := range s.WorkHistory {
					if hexutil.MustDecodeUint64(work[3])+staleThreshold <= hexutil.MustDecodeUint64(s.LastWork[3]) {
						delete(s.WorkHistory, hash)
					}
				}
			}
		}
	}
}

func (s *Sender) Attach(subs ...Subscriber) {
	s.subs = append(s.subs, subs...)
}

func (s *Sender) Detach(sub Subscriber) {
	for k, v := range s.subs {
		if sub == v {
			s.subs = append(s.subs[:k], s.subs[k+1:]...)
		}
	}
}

func (s *Sender) Close() {
	close(s.quit)

	s.wg.Wait()
}

// GetLastWork 获取最后任务
func (s *Sender) GetLastWork() []string {
	if s.LastWork != nil {
		work := make([]string, len(s.LastWork))
		copy(work, s.LastWork)
		work[2] = s.WorkTarget
		return work
	}

	return nil
}

// GetWorkByHeader 获取任务历史
func (s *Sender) GetWorkByHeader(header string) ([]string, bool) {
	val, ok := s.WorkHistory[header]
	return val, ok
}
