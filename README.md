# CDN Acceptance Tests

Acceptance tests for our Content Delivery Network(s).

These are written using Go's [testing][testing] package because it provides
a framework for running basic assertions and a [rich HTTP client/server
library][net/http].

[testing]: http://golang.org/pkg/testing/
[net/http]: http://golang.org/pkg/net/http/

## Methodology

The single Go process acts as both the client and the origin server so that
it can inspect the input and output of the CDN.
```
              +---------+
        +---> |   CDN   |-----+
        |     +---------+     |
 client |                     | server
        |     +---------+     |
        +-----| go test | <---+
              +---------+
```

It will not configure the CDN service for you; you'll need to do so,
pointing it at the machine that will be running the tests.

## Running

You will need the Go 1.x runtime installed. To install this on OS X:
```sh
brew install go
```

To run the tests:
```sh
go test
```

## Mock CDN virtual machine

You can use the included Vagrant VM, which runs Nginx and Varnish, to mock
CDN behaviour. This can be useful when developing new tests or
functionality, while either offline or in parallel to someone else.

It is unlikely that *all* tests will run successfully. If you want a
particular test to pass you may need to modify the Nginx or Varnish configs
in [`mock_cdn_config/`](/mock_cdn_config) accordingly.

To bring up the VM and point the tests at it:
```
vagrant up && vagrant provision
go test -edgeHost 172.16.20.10 -insecureTLS
```

Please note that this is not a complete substitute for the real thing. You
**must** test against a real CDN before submitting any pull requests.
