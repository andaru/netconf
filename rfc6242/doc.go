// Copyright (c) 2018 Andrew Fort
//

// Package rfc6242 provides support for NETCONF over SSH transports.
//
// RFC6242 defines the NETCONF-over-SSH transport, both the SSH2 protocol
// details as well as a transport or wire protocol used also by
// NETCONF-over-TLS (RFC5539).
//
// This package supports both "end of message delimited" (:base:1.0) and
// "chunked framing" (:base:1.1) modes. NETCONF sessions can upgrade a Decoder
// and Encoder's framing mode depending on session capability negotiation.
//
// For writing to NETCONF peers, the Encoder provides the interface to
// the NETCONF messages layer, offering io.Writer and an EndOfMessage
// method used to signal the end of each message in the NETCONF stream.
//
// To decode a NETCONF message stream from a peer, the Decoder type
// implements io.Reader and zero-copy io.WriterTo support.
package rfc6242
