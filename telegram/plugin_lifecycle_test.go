package telegram

import (
	"context"
	"testing"
	"time"
)

type reentrantStopPlugin struct {
	client *Client
}

func (*reentrantStopPlugin) Name() string { return "reentrant-stop" }

func (*reentrantStopPlugin) Start(context.Context, *Client) error { return nil }

func (p *reentrantStopPlugin) Stop(context.Context) error {
	p.client.Use(p)
	return nil
}

func TestPluginStopRunsOutsideClientLock(t *testing.T) {
	client, _ := NewClient(1, "hash", nil)
	plugin := &reentrantStopPlugin{client: client}
	client.Use(plugin)
	done := make(chan struct{})
	go func() {
		client.stopPlugins(context.Background())
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Plugin.Stop deadlocked while calling a client method")
	}
}
