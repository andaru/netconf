package schema

import (
	"context"
	"encoding/xml"

	"github.com/andaru/netconf/rfc6242"
	"github.com/golang/glog"
)

type serverSessionParser struct {
	input   *rfc6242.Decoder
	output  *rfc6242.Encoder
	decoder *xml.Decoder
	cancel  context.CancelFunc

	chunkedMode bool
}

func (x *serverSessionParser) Init() error {
	glog.V(1).Info("INIT")
	return nil
}

func (x *serverSessionParser) RemoteCapabilities(caps []string) error {
	for _, cap := range caps {
		switch cap {
		case capBase10:
		case capBase11:
		}
		if cap == capBase11 {
			glog.V(1).Infof("GOT BASE11")
			x.chunkedMode = true
		}
	}
	glog.V(1).Infof("REMOTE_CAPABILITIES %#v: %#v", caps, x)
	return nil
}

func (x *serverSessionParser) RemoteSessionID(id string) error {
	// it is an error to receive the /hello/session-id element
	// from a NETCONF client
	x.cancel()
	glog.V(1).Infof("REMOTE_SESSION_ID %q", id)
	return nil
}

func (x *serverSessionParser) HelloReceived() error {
	glog.V(1).Info("HELLO_RECEIVED")
	if x.chunkedMode {
		glog.V(1).Info("BASE11_CHUNKED_FRAMING")
		rfc6242.SetChunkedFraming(x.input, x.output)
	}
	return nil
}

func (x *serverSessionParser) RPC(e xml.StartElement) error {
	glog.V(1).Infof("RPC %s", pprintElem(e))
	return nil
}

func (x *serverSessionParser) RPCReply(e xml.StartElement) error { return nil }

func (x *serverSessionParser) Notification(e xml.StartElement) error { return nil }

func (x *serverSessionParser) Finished(err error) {
	glog.V(1).Infof("FINISHED %#v", err)
}
