/*
Package transport provides the NETCONF transport layer.

The transport layer provides "clean" Reader and Writer
interfaces to respectively decode and encode traffic
for the underlying transport layer.  The message layer
reads and writes to these transport layer objects.
*/
package transport
