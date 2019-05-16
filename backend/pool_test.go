package backend

import (
	"errors"
	"net"
	"testing"
	"time"
)

// Errors
var ErrTestConnectionCreation = errors.New("connection creation error")
var ErrTestClose = errors.New("close error")
var ErrTestSetWriteDeadline = errors.New("set write deadline error")

type TestConn struct {
	failOnSetWriteDeadline bool
	failOnClose            bool
}

func (t TestConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (t TestConn) Write(b []byte) (n int, err error) {
	return 0, nil
}

func (t TestConn) Close() error {
	if t.failOnClose {
		return ErrTestClose
	}
	return nil
}

func (t TestConn) LocalAddr() net.Addr {
	return nil
}

func (t TestConn) RemoteAddr() net.Addr {
	return nil
}

func (t TestConn) SetDeadline(ti time.Time) error {
	return nil
}

func (t TestConn) SetReadDeadline(ti time.Time) error {
	return nil
}

func (t TestConn) SetWriteDeadline(ti time.Time) error {
	if t.failOnSetWriteDeadline {
		return ErrTestSetWriteDeadline
	}
	return nil
}

func testMkGoodConn(host string) (net.Conn, error) {
	return &TestConn{failOnSetWriteDeadline: false, failOnClose: false}, nil
}

func testMkConnSetDeadlineFailure(host string) (net.Conn, error) {
	return &TestConn{failOnSetWriteDeadline: true, failOnClose: false}, nil
}

func testMkConnCloseFailure(host string) (net.Conn, error) {
	return &TestConn{failOnSetWriteDeadline: false, failOnClose: true}, nil
}

func testMkConnFailure(host string) (net.Conn, error) {
	return nil, ErrTestConnectionCreation
}
func TestConnPool(t *testing.T) {
	cp := NewPool("somehost")
	cp.mkConn = testMkGoodConn
	seenConns := map[net.Conn]bool{}

	// able to get upto maxconn+maxoverflow
	for i := 0; i < maxConn+maxOverflow; i++ {
		sc, err := cp.Get()
		if err != nil {
			t.Fatalf("Error getting connection from pool: %v", err)
		}
		seenConns[sc] = true
	}
	// connection pool should be empty now and overflow should be maxxed out
	assertConnPoolState(cp, t, 0, maxConn+maxOverflow)

	// trying to get more connection should fail
	_, err := cp.Get()
	if ErrTimeout != err {
		t.Errorf("Expected %v but got %v", ErrTimeout, err)
	}
	assertConnPoolState(cp, t, 0, maxConn+maxOverflow)

	// releasing all acquired connections should fill up the connection pool
	for k := range seenConns {
		cp.Return(k, false)
	}
	assertConnPoolState(cp, t, maxConn, maxOverflow)

	// connections should now be reused
	reusedConn, err := cp.Get()
	if err != nil {
		t.Fatalf("Error getting connection from pool: %v", err)
	}
	if _, exists := seenConns[reusedConn]; !exists {
		t.Fatalf("Was expecting connection reuse")
	}
	assertConnPoolState(cp, t, maxConn-1, maxOverflow)
}

func assertConnPoolState(cp *Pool, t *testing.T, expectedPoolCount int, expectedSemCount int) {
	if (len(cp.connections) != expectedPoolCount) || (len(cp.createsem) != expectedSemCount) {
		t.Fatalf("expected %v connections in the pool and %v as the semaphoreCount, but got %v and %v respectively",
			expectedPoolCount, expectedSemCount, len(cp.connections), len(cp.createsem))
	}
}

func assertErrorType(t *testing.T, expectedError error, gotError error) {
	if expectedError != gotError {
		t.Fatalf("was expecting %v but got %v", expectedError, gotError)
	}
}

func TestConnPoolFailures(t *testing.T) {
	cp := NewPool("somehost")

	cp.mkConn = testMkConnFailure
	_, err := cp.Get()
	assertErrorType(t, ErrTestConnectionCreation, err)
	assertConnPoolState(cp, t, 0, 0)

	cp.mkConn = testMkConnSetDeadlineFailure
	_, err = cp.Get()
	assertErrorType(t, ErrTestSetWriteDeadline, err)
	assertConnPoolState(cp, t, 0, 0)

	cp.mkConn = testMkConnCloseFailure
	conn, err := cp.Get()
	assertErrorType(t, nil, err)
	assertConnPoolState(cp, t, 0, 1)
	cp.Return(conn, false)
	assertConnPoolState(cp, t, 1, 1)

	//
	cp.mkConn = testMkGoodConn
	conn, err = cp.Get()
	assertErrorType(t, nil, nil)
	assertConnPoolState(cp, t, 0, 1)
	cp.Return(conn, true)
	assertConnPoolState(cp, t, 0, 0)

}
