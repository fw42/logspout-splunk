package splunk

import (
	"fmt"
	"net"

	"github.com/gliderlabs/logspout/router"
)

func init() {
	router.AdapterFactories.Register(NewSplunkAdapter, "splunk")
}

type SplunkAdapter struct {
	address    *net.TCPAddr
	connection *net.TCPConn
	queue      chan *router.Message
	route      *router.Route
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

	adapter := &SplunkAdapter{
		address: address,
		route:   route,
		queue:   queue,
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

func (splunk *SplunkAdapter) reconnect() error {
	splunk.disconnect()

	var err error

	for tries := 0; tries < 3; tries++ {
		err = splunk.connect()

		if err == nil {
			return nil
		}
	}

	return err
}

func (splunk *SplunkAdapter) writeData(b []byte) {
	for {
		bytesWritten, err := splunk.connection.Write(b)
		if err != nil {
			fmt.Printf("Failed to write to TCP connection: %s\n", err)
			fmt.Printf("Reconnecting...\n")
			err = splunk.reconnect()
			break
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
			fmt.Printf("woot channel is full")
			splunk.route.Close()
			return
		}
	}
}
