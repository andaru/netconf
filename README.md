# NETCONF support libraries for golang #

This repository offers support libraries for writing
[NETCONF](https://tools.ietf.org/html/rfc6241) applications in
[Go](https://golang.org/).

  * [RFC6242 transport](https://github.com/andaru/netconf/tree/master/rfc6242) decoding and encoding filters
    * Support for both [end-of-message](https://tools.ietf.org/html/rfc6242#section-4.3) and [chunked-message](https://tools.ietf.org/html/rfc6242#section-4.2) encoding
  * A gRPC based bi-directional transport stream, to use an OpenSSH
    server with a NETCONF agent
  * NETCONF XML schema and state machines, used to manage XML stream decoding
	* Attach callback functions to the <rpc> or <rpc-reply> schema nodes
      to build NETCONF server and client applications
  * Idiomatic use of `io` interfaces throughout

Libraries under development:

  * A document object model
  * Encoding, decoding and validation of YANG data, including:
    * RFC6020 XML encoding and decoding of YANG data
	* RFC7951 JSON encoding and decoding of YANG data
	* Generation of NETCONF <rpc> objects from YANG `rpc` statements

Applications you can build with these libraries include:

  * NETCONF clients, servers
  * NETCONF to network-element CLI proxies for non-NETCONF devices
  * NETCONF/YANG command-line tools for network operator use

For building applications with an awareness of
[YANG](https://tools.ietf.org/html/rfc7950) data used with NETCONF
services, see the [goyang](https://github.com/openconfig/goyang)
library.
