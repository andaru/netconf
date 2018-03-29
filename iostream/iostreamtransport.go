package iostream

import (
	"io"
	"sync"
)

// sendRecvIntf are the write/read calls which exist on both the
// IOStream client and server interfaces. This abstraction permits an
// *iostreamTransport to be used in both a client or a server.
type sendRecvIntf interface {
	Send(*TransportPDU) error
	Recv() (*TransportPDU, error)
}

type closeSender interface {
	CloseSend() error
}

type iostreamTransport struct {
	channelID int32
	stream    sendRecvIntf
	pr        *io.PipeReader
	pw        *io.PipeWriter
	wn        int64

	mu sync.Mutex
}

func newIOStreamTransport(id int32, stream sendRecvIntf) *iostreamTransport {
	s := &iostreamTransport{channelID: id, stream: stream}
	s.pr, s.pw = io.Pipe()
	return s
}

func (s *iostreamTransport) Read(b []byte) (n int, err error) {
	return s.pr.Read(b)
}

func (s *iostreamTransport) Write(b []byte) (n int, err error) {
	s.mu.Lock()
	tpdu := &TransportPDU{ChannelID: s.channelID, Data: b, DataType: TransportPDU_DATA_TX}
	if err = s.stream.Send(tpdu); err == nil {
		s.wn += int64(len(b))
	}
	s.mu.Unlock()
	return
}

func (s *iostreamTransport) Close() error {
	s.pw.Close()
	s.pr.Close()
	return nil
}

func (s *iostreamTransport) CloseSend() (err error) {
	if err = s.stream.Send(&TransportPDU{
		ChannelID: s.channelID, DataType: TransportPDU_DATA_TX_CLOSE}); err != nil {
		return
	}

	if csender, ok := s.stream.(closeSender); ok {
		err = csender.CloseSend()
	}
	return
}
