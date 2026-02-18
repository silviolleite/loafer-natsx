package conn_test

import (
	"net"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	natstest "github.com/nats-io/nats-server/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/silviolleite/loafer-natsx/conn"
)

func runServer() (*server.Server, string) {
	opts := natstest.DefaultTestOptions
	opts.Port = -1
	s := natstest.RunServer(&opts)
	return s, s.ClientURL()
}

func TestConnect_Success_Defaults(t *testing.T) {
	s, url := runServer()
	defer s.Shutdown()

	nc, err := conn.Connect(url)
	assert.NoError(t, err)
	assert.NotNil(t, nc)
	assert.True(t, nc.IsConnected())

	nc.Close()
}

func TestConnect_InvalidURL(t *testing.T) {
	nc, err := conn.Connect("nats://invalid:4222", conn.WithTimeout(100*time.Millisecond))
	assert.Error(t, err)
	assert.Nil(t, nc)
}

func TestConnect_WithName(t *testing.T) {
	s, url := runServer()
	defer s.Shutdown()

	nc, err := conn.Connect(url, conn.WithName("test-connection"))
	assert.NoError(t, err)
	assert.NotNil(t, nc)
	assert.Equal(t, "test-connection", nc.Opts.Name)

	nc.Close()
}

func TestConnect_WithReconnectOptions(t *testing.T) {
	s, url := runServer()
	defer s.Shutdown()

	nc, err := conn.Connect(
		url,
		conn.WithReconnectWait(500*time.Millisecond),
		conn.WithMaxReconnects(5),
	)
	assert.NoError(t, err)
	assert.NotNil(t, nc)
	assert.Equal(t, 5, nc.Opts.MaxReconnect)
	assert.Equal(t, 500*time.Millisecond, nc.Opts.ReconnectWait)

	nc.Close()
}

func TestConnect_WithTimeout(t *testing.T) {
	s, url := runServer()
	defer s.Shutdown()

	nc, err := conn.Connect(
		url,
		conn.WithTimeout(2*time.Second),
	)
	assert.NoError(t, err)
	assert.NotNil(t, nc)
	assert.Equal(t, 2*time.Second, nc.Opts.Timeout)

	nc.Close()
}

func TestConnect_MultipleOptions(t *testing.T) {
	s, url := runServer()
	defer s.Shutdown()

	nc, err := conn.Connect(
		url,
		conn.WithName("multi"),
		conn.WithReconnectWait(200*time.Millisecond),
		conn.WithMaxReconnects(3),
		conn.WithTimeout(3*time.Second),
	)
	assert.NoError(t, err)
	assert.NotNil(t, nc)

	assert.Equal(t, "multi", nc.Opts.Name)
	assert.Equal(t, 200*time.Millisecond, nc.Opts.ReconnectWait)
	assert.Equal(t, 3, nc.Opts.MaxReconnect)
	assert.Equal(t, 3*time.Second, nc.Opts.Timeout)

	nc.Close()
}

func TestConnect_ServerDownAfterConnect(t *testing.T) {
	s, url := runServer()

	nc, err := conn.Connect(url)
	assert.NoError(t, err)
	assert.True(t, nc.IsConnected())

	s.Shutdown()
	time.Sleep(100 * time.Millisecond)

	assert.True(t, nc.IsClosed() || !nc.IsConnected())
}

func TestConnect_UnreachablePort(t *testing.T) {
	l, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)
	addr := l.Addr().String()
	l.Close()

	nc, err := conn.Connect("nats://"+addr, conn.WithTimeout(100*time.Millisecond))
	assert.Error(t, err)
	assert.Nil(t, nc)
}
