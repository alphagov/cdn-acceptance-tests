# CDN Acceptance Tests

Acceptance tests for our Content Delivery Network(s).

These are written using Go's [testing][testing] package because it provides
a framework for running basic assertions and allows us to construct HTTP
clients and servers from [net/http][net/http].

[testing]: http://golang.org/pkg/testing/
[net/http]: http://golang.org/pkg/net/http/

## Running

You will need the Go 1.x runtime installed. To install this on OS X:
```sh
brew install go
```

To run the tests:
```sh
go test
```
