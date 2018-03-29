package iostream

import (
	"bytes"
	"io"
	"testing"

	"net"

	"github.com/stretchr/testify/assert"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
)

func TestIOStream(t *testing.T) {
	check := assert.New(t)

	srv := grpc.NewServer()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		if err := srv.Serve(l); err != nil {
			t.Fatal(err)
		}
	}()

	wantMsg := "Hello, world!\n"

	acceptor := func(ctx context.Context, transport TransportStream) (_ context.CancelFunc, err error) {
		go func() {
			transport.Write([]byte(wantMsg))
			transport.Write([]byte(wantMsg))
			transport.CloseSend()
		}()

		return
	}

	RegisterIOStreamServer(srv, &Service{AcceptTransport: acceptor})

	conn, err := grpc.Dial(l.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	client := NewClient(conn)
	defer conn.Close()
	defer client.Close()

	clientCh := make(chan error, 1)
	go func(ctx context.Context) {
		clientCh <- client.Run(ctx)
		close(clientCh)
	}(context.Background())

	buf := &bytes.Buffer{}
	io.Copy(buf, client)

	select {
	case err = <-clientCh:
		check.NoError(err)
	default:
	}

	check.Equal(wantMsg+wantMsg, buf.String())
}
