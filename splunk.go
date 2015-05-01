package splunk

import (
	"fmt"
	"github.com/gliderlabs/logspout/router"
	"net"
	"time"
)

func init() {
	router.AdapterFactories.Register(NewSplunkAdapter, "splunk")
}

type SplunkAdapter struct {
	address    *net.TCPAddr
	connection *net.TCPConn
	queue      chan *router.Message
	route      *router.Route
	done       chan bool
}

func NewSplunkAdapter(route *router.Route) (router.LogAdapter, error) {
	addrStr := route.Address
	if len(addrStr) == 0 {
		return nil, fmt.Errorf("Address missing")
	}

	address, err := net.ResolveTCPAddr("tcp", addrStr)
	if err != nil {
		return nil, err
	}

	queue := make(chan *router.Message, 1024)
	done := make(chan bool, 1)

	adapter := &SplunkAdapter{
		address: address,
		route:   route,
		queue:   queue,
		done:    done,
	}

	if err = adapter.connect(); err != nil {
		return nil, err
	}

	go adapter.writer()
	return adapter, nil
}

func (splunk *SplunkAdapter) connect() error {
	connection, err := net.DialTCP("tcp", nil, splunk.address)
	if err != nil {
		return err
	}

	if err = connection.SetKeepAlive(true); err != nil {
		return err
	}

	splunk.connection = connection

	return nil
}

func (splunk *SplunkAdapter) disconnect() error {
	if splunk.connection == nil {
		return nil
	}

	return splunk.connection.Close()
}

func (splunk *SplunkAdapter) reconnectLoop() {
	splunk.disconnect()

	var err error

	for {
		select {
		case <-splunk.done:
			break
		default:
		}

		err = splunk.connect()
		if err == nil {
			break
		}

		fmt.Printf("Splunk reconnect failed: %s\n", err)
		time.Sleep(1 * time.Second)
	}
}

func (splunk *SplunkAdapter) writeData(b []byte) {
	for {
		bytesWritten, err := splunk.connection.Write(b)

		if err != nil {
			fmt.Printf("Failed to write to TCP connection: %s\n", err)
			splunk.reconnectLoop()
			return
		}

		fmt.Printf("Wrote %v...", string(b))
		b = b[bytesWritten:]

		if len(b) == 0 {
			break
		}
	}
}

func (splunk *SplunkAdapter) writer() {
	for message := range splunk.queue {
		splunk.writeData([]byte(message.Data + "\n"))
	}
}

func (splunk *SplunkAdapter) Stream(logstream chan *router.Message) {
	for message := range logstream {
		select {
		case splunk.queue <- message:
		default:
			fmt.Printf("Channel is full! Dropping events :-(")
			continue
		}
	}

	splunk.Close()
}

func (splunk *SplunkAdapter) Close() {
	close(splunk.queue)
	splunk.disconnect()
	splunk.done <- true
}
