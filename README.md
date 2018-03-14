# NETCONF support libraries for golang #

This repository offers support libraries for writing
[NETCONF](https://tools.ietf.org/html/rfc6241) applications in
[Go](https://golang.org/).

  * [RFC6242 transport](https://github.com/andaru/netconf/tree/master/rfc6242) decoding and encoding filters.
    * Support for both [end-of-message](https://tools.ietf.org/html/rfc6242#section-4.3) and [chunked-message](https://tools.ietf.org/html/rfc6242#section-4.2) encoding.

For building NETCONF applications with awareness of
[YANG](https://tools.ietf.org/html/rfc7950) data, see the
[goyang](https://github.com/openconfig/goyang) library.

