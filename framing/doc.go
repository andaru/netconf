/*
Package framing offers RFC6242 end-of-message and chunked framing decoders.

The functions in this package return bufio.SplitFunc for use with a *bufio.Scanner.
These functions will return io.ErrUnexpectedEOF when input terminates other
than at the end of a message.
*/
package framing
