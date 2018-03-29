package iostream

import (
	"io"

	context "golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service provides the gRPC service iostream.IOStream
type Service struct {
	// AcceptTransport is the transport stream accept callback.
	//
	// This function is called by IOStream service at the beginning of
	// calls to the TransportStream RPC with the gRPC stream context
	// and the transport stream object. When the function returns, the
	// service will read from the stream and make this data available
	// to the transport stream's io.Reader.
	//
	// If shutdown is non-nil, it is called when the server stream
	// handler finishes. This function should be used to close and
	// release all resources held by the acceptor, such as IO handles
	// and goroutines.
	//
	// In addition, the context can be monitored for completion, which
	// indicates gRPC has closed the server stream.
	AcceptTransport func(context.Context, TransportStream) (shutdown context.CancelFunc, err error)
}

// TransportStream is the transport stream type offered by the Service
// to the AcceptTransport function.
//
// TransportStream typically have their io.Reader attached to data
// from the gRPC TransportStream stream and an io.Writer which sends
// data to the stream.  Calling Close will shut down the send and
// receive sides of the transport, while calling CloseSend closes just
// the write half, signaling EOF to the peer.
type TransportStream interface {
	io.ReadWriteCloser
	CloseSend() error
}

// TransportStream is the RPC method to create a bi-directional, multi-channel data stream.
func (svc *Service) TransportStream(stream IOStream_TransportStreamServer) (err error) {
	transport := newIOStreamTransport(1, stream)
	defer transport.Close()
	cancel, err := svc.AcceptTransport(stream.Context(), transport)
	if err != nil {
		err = status.Errorf(codes.PermissionDenied, "%s", err)
	}
	if cancel != nil {
		defer func() {
			cancel()
		}()
	}

	// send input from the stream to the transport
	ch := make(chan error, 1)
	var tpdu *TransportPDU
	go func() {
		defer close(ch)
		for {
			tpdu, err = stream.Recv()
			if err != nil {
				ch <- err
				return
			}
			if tpdu.GetChannelID() != ioChannel {
				continue
			}
			switch tpdu.GetDataType() {
			case TransportPDU_DATA_TX:
				if data := tpdu.GetData(); len(data) > 0 {
					_, err = transport.Write(data)
				}
			case TransportPDU_DATA_TX_CLOSE:
				if data := tpdu.GetData(); len(data) > 0 {
					_, err = transport.Write(data)
				}
				transport.pw.Close()
			}

			if err != nil {
				ch <- err
				return
			}
		}
	}()

	if err = <-ch; err == io.EOF {
		err = nil
	}

	if err != nil {
		if _, ok := status.FromError(err); !ok {
			err = status.Errorf(codes.Internal, "%s", err)
		}
	}
	return
}
