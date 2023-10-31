package notifications

import (
	"sync"

	"github.com/golden-vcr/ledger"
)

type subscriberChannels struct {
	chans map[string][]chan *ledger.Transaction
	mu    sync.RWMutex
}

func (s *subscriberChannels) register(twitchUserId string) chan *ledger.Transaction {
	ch := make(chan *ledger.Transaction, 32)
	s.mu.Lock()
	defer s.mu.Unlock()

	s.chans[twitchUserId] = append(s.chans[twitchUserId], ch)
	return ch
}

func (s *subscriberChannels) unregister(twitchUserId string, ch chan *ledger.Transaction) {
	s.mu.Lock()
	defer s.mu.Unlock()

	chs, ok := s.chans[twitchUserId]
	if ok {
		for i := 0; i < len(chs); i++ {
			if chs[i] == ch {
				head := chs[:i]
				tail := chs[i+1:]
				s.chans[twitchUserId] = append(head, tail...)
				return
			}
		}
	}
}

func (s *subscriberChannels) broadcast(twitchUserId string, transaction *ledger.Transaction) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chs, ok := s.chans[twitchUserId]
	if ok {
		for _, ch := range chs {
			ch <- transaction
		}
	}
}
