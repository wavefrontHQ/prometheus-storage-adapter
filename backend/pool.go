package backend

import (
	"errors"
	"io"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

const maxConn int = 10
const maxOverflow int = 10
const maxConnWait time.Duration = 10 * time.Millisecond

// Errors
var ErrTimeout = errors.New("timeout waiting to build connection")

type Pool struct {
	host        string
	connections chan (net.Conn)
	createsem   chan (bool)
	mkConn      func(host string) (net.Conn, error)
}

func NewPool(host string) *Pool {
	return &Pool{
		host:        host,
		connections: make(chan (net.Conn), maxConn),
		createsem:   make(chan (bool), maxConn+maxOverflow),
		mkConn:      defaultMkConn,
	}
}

func defaultMkConn(host string) (net.Conn, error) {
	return net.Dial("tcp", host)
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
			conn, err := cp.mkConn(cp.host)
			if err != nil {
				// On error, release our create hold
				cp.release(conn)
				return nil, err
			}
			conn, err = prepareConnection(conn)
			if err != nil {
				// On error, release our create hold
				cp.release(conn)
				return nil, err
			}
			return conn, err
		case <-time.After(maxConnWait):
			log.Debugf("Max connection exceeded")
			return nil, ErrTimeout
		}
	}
}

func (cp *Pool) release(conn net.Conn) {
	<-cp.createsem
	if conn != nil {
		conn.Close()
	}
}

func (cp *Pool) connOk(conn net.Conn) bool {
	if conn == nil {
		return false
	}
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Millisecond)); err != nil {
		return false
	}
	// you have to try reading atleast 1 byte to detect closed connection
	b1 := make([]byte, 1)
	_, err := conn.Read(b1)
	if err != nil && err == io.EOF {
		log.Info("Connection is closed")
		return false
	}
	return true
}

func (cp *Pool) Return(conn net.Conn, failed bool) {
	if failed || !cp.connOk(conn) {
		cp.release(conn)
		return
	}
	select {
	case cp.connections <- conn:
	default:
		// Overflow connection.
		cp.release(conn)
	}
}
