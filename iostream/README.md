# iostream: gRPC transport aggregation proxy #

The `iostream` package provides a gRPC transport stream service and a corresponding client.

The service can be used to connect a client, such as a third-party SSH
or TLS daemon, to a NETCONF server. In the case of NETCONF-over-SSH
and OpenSSH, a subsystem (such as that for `Subsystem netconf`) would
act as the transport stream client and the NETCONF agent process acts
as the transport stream server.

## Rebuild generated protocol buffer code and gRPC stubs ##

```
GOPATH=$(go env GOPATH)
cd ${GOPATH}/src/github.com/andaru/netconf/iostream
protoc -I=${GOPATH}/src/github.com/andaru/netconf/iostream --go_out=plugins=grpc:. iostream.proto
```
