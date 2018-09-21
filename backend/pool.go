package backend

import (
	"net"
	"time"

	log "github.com/Sirupsen/logrus"
)

const maxConn int = 10

type Pool struct {
	host        string
	connections chan (net.Conn)
	createsem   chan (bool)
}

func NewPool(host string) *Pool {
	return &Pool{
		host:        host,
		connections: make(chan (net.Conn), maxConn),
		createsem:   make(chan (bool), 1),
	}
}

func prepareConnection(conn net.Conn) (net.Conn, error) {
	if err := conn.SetWriteDeadline(time.Now().Add(60 * time.Second)); err != nil {
		return nil, err
	}
	return conn, nil
}

// Based on an algorithm by Dustin Sallins:
// http://dustin.sallings.org/2014/04/25/chan-pool.html
func (cp *Pool) Get() (net.Conn, error) {
	log.Debugf("Trying to get connection")
	// Try to grab an available connection within 1ms
	select {
	case conn := <-cp.connections:
		return prepareConnection(conn)
	case <-time.After(time.Millisecond):
		// No connection came around in time, let's see
		// whether we can get one or build a new one first.
		log.Debugf("No connection in pool")
		select {
		case conn := <-cp.connections:
			return prepareConnection(conn)
		case cp.createsem <- true:
			// Room to make a connection
			log.Debugf("About to connect")
			conn, err := net.Dial("tcp", cp.host)
			if err != nil {
				// On error, release our create hold
				<-cp.createsem
				return nil, err
			}
			return prepareConnection(conn)
		}
	}
}

func (cp *Pool) Return(c net.Conn) {
	select {
	case cp.connections <- c:
	default:
		// Overflow connection.
		<-cp.createsem
		c.Close()
	}
}
