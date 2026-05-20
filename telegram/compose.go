package telegram

import (
	"fmt"
	"sync"
)

var (
	globalClientsMu sync.Mutex
	globalClients   []*Client
)

func registerClient(c *Client) {
	globalClientsMu.Lock()
	globalClients = append(globalClients, c)
	globalClientsMu.Unlock()
}

// Idle blocks the calling goroutine until all registered clients have been
// stopped. This is the package-level equivalent of calling Client.Idle() on
// every active client.
//
// Example:
//
//	client1, _ := telegram.NewClient(telegram.ClientConfig{...})
//	client2, _ := telegram.NewClient(telegram.ClientConfig{...})
//	go client1.Start()
//	go client2.Start()
//	telegram.Idle()
func Idle() {
	globalClientsMu.Lock()
	clients := make([]*Client, len(globalClients))
	copy(clients, globalClients)
	globalClientsMu.Unlock()

	var wg sync.WaitGroup
	for _, c := range clients {
		wg.Add(1)
		go func(cl *Client) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					cl.Log.Errorf("Idle goroutine panic: %v", r)
				}
			}()
			cl.Idle()
		}(c)
	}
	wg.Wait()
}

// Compose starts multiple clients concurrently and blocks until all of them
// have stopped. Each client is started in its own goroutine. If any client
// fails to start, the already-started clients are stopped and an error is
// returned.
//
// Example:
//
//	err := telegram.Compose(client1, client2, client3)
//	if err != nil {
//	    log.Fatal(err)
//	}
func Compose(clients ...*Client) error {
	var (
		wg     sync.WaitGroup
		errCh  = make(chan error, len(clients))
		stopCh = make(chan struct{})
		first  error
	)

	for _, c := range clients {
		wg.Add(1)
		go func(cl *Client) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					err := fmt.Errorf("Compose goroutine panic: %v", r)
					cl.Log.Errorf("%v", err)
					select {
					case errCh <- err:
					default:
					}
				}
			}()
			if err := cl.Start(); err != nil {
				select {
				case errCh <- err:
				default:
				}
			}
		}(c)
	}

	go func() {
		wg.Wait()
		close(stopCh)
	}()

	select {
	case first = <-errCh:
		for _, c := range clients {
			go c.Stop()
		}
		<-stopCh
	case <-stopCh:
	}

	return first
}
