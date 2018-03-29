package iostream

import (
	"errors"
	"io"

	"github.com/golang/glog"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	ioChannel  = int32(1)
	errChannel = int32(2)
)

// Client is a gRPC IOStream client, implementing TransportStream.
type Client struct {
	Conn *grpc.ClientConn

	io    *iostreamTransport
	start chan struct{}
	pwN   int64 // bytes made available for reading
	prN   int64 // bytes read
	wN    int64 // bytes of data written to the server (payload; not including gRPC framing)
}

// NewClient returns a client IOStream with the gRPC connection conn.
//
// The returned *Client implements the io.ReadWriteCloser interface
// along with CloseSend for closing the write side of the stream only.
//
// Once created, calls to Read, Write, Close and CloseSend will block
// until the transport is started by calling Run().  If Run() returns
// in error, any blocked calls to the above functions may return lower
// level transport related errors, such as io.EOF or io.ErrClosedPipe.
func NewClient(conn *grpc.ClientConn) *Client {
	return &Client{Conn: conn, start: make(chan struct{})}
}

func (c *Client) Read(b []byte) (int, error) {
	<-c.start
	n, err := c.io.Read(b)
	c.prN += int64(n)
	return n, err
}

func (c *Client) Write(b []byte) (int, error) {
	<-c.start
	n, err := c.io.Write(b)
	c.wN += int64(n)
	return n, err
}

// Close closes the IO stream
func (c *Client) Close() error {
	<-c.start
	return c.io.Close()
}

// CloseSend closes only the write side of the IO stream
func (c *Client) CloseSend() error {
	<-c.start
	return c.io.CloseSend()
}

// Run starts and runs a client transport stream until it completes.
//
// This function starts a gRPC TransportStream, attaches the stream to
// the type's io.ReadWriteCloser interface and then passes data until
// the gRPC transport stream finishes. The running stream responds to
// CloseSend() by closing the stream for writing and sending EOF to
// the server.
//
// Because calls to this type's io.ReadWriteCloser interface will also
// block until this function is called, this function should be called
// concurrently with any IO operations.
func (c *Client) Run(ctx context.Context) (err error) {
	if c.io != nil {
		panic("Client.Run already called")
	} else if c.Conn == nil {
		err = errors.New("Client.Conn must not be nil")
	} else if ctx.Err() != nil {
		err = ctx.Err()
	}

	var stream IOStream_TransportStreamClient
	var client IOStreamClient
	// create a stub for the IOStream service and call the TransportStream RPC
	if err == nil {
		client = NewIOStreamClient(c.Conn)
		stream, err = client.TransportStream(ctx)
	}

	// bind our io.ReadWriteCloser interfaces to the gRPC stream
	if err == nil {
		c.io = newIOStreamTransport(1, stream)
		defer c.io.Close()
	}

	// signal IO function readiness to io.ReadWriteCloser interface.
	close(c.start)
	if err != nil {
		return err
	}

	// make stream input (the client's "stdout") available for Read()
	var tpdu *TransportPDU
	var n int
	for {
		tpdu, err = stream.Recv()
		switch {
		case err != nil:
			glog.V(9).Infof("stream.Recv() err=%#v", err)
		case ctx.Err() != nil:
			err = ctx.Err()
		case tpdu.GetChannelID() != ioChannel:
			// ignore data on an unexpected channel
		case tpdu.GetDataType() == TransportPDU_DATA_TX:
			if data := tpdu.GetData(); len(data) > 0 {
				// received a DATA frame, write it to the pipe for calls to Read
				n, err = c.io.pw.Write(data)
				c.pwN += int64(n)
			}
		case tpdu.GetDataType() == TransportPDU_DATA_TX_CLOSE:
			// received a final DATA frame with an EOF indication. write it to
			// the pipe and close the transport.
			if data := tpdu.GetData(); len(data) > 0 {
				n, err = c.io.pw.Write(data)
				c.pwN += int64(n)
			}
			// close the write side of our pipe so that Read() returns EOF.
			c.io.pw.Close()
			if err == nil {
				err = io.EOF
			}
		}

		if err == io.EOF {
			err = nil
			break
		} else if err != nil {
			break
		}
	}

	return
}
