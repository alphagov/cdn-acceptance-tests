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
         +-------> |         |---------+
         |         |   CDN   |         |
         | +-------|         | <-----+ |
         | |       +---------+       | |
         | |                         | |
 request-| |-response                | |
         | |                         | |
         | |   +-----------------+   | |
         | +-> |     go test     |---+ |
         |     |                 |     |
         +-----| client ¦ server | <---+
               +-----------------+
```

When testing a real CDN, the tests must be run on a server that the CDN can
connect to.

It will not configure the CDN service for you; you'll need to do so,
pointing it at the machine that will be running the tests.

## Running

You will need the Go 1.x runtime installed. To install this on OS X:
```sh
brew install go
```

To run all the tests:
```sh
go test
```

To run a subset of tests based on a regex:
```sh
go test -run 'Test(Cache|NoCache)'
```

To see all available command-line options:
```sh
go test -usage
```

## Writing tests

When writing new tests please be sure to:

- group the test in a file with other tests of similar behaviour e.g.
  "custom failover"
- use a consistent naming prefix for the functions that so that they can be
  run as a group e.g. `func TestCustomFailover…(…)`
- always call `ResetBackendsInOrder()` at the beginning of each test to
  ensure that all of the backends are running and have their handlers reset
  from previous tests.
- use the helpers such as `NewUniqueEdgeGET()` and `RoundTripCheckError()`
  which do a lot of the work, such as error checking, for you.
- define static inputs such as "number of requests" or "time between
  requests" at the beginning of the test so that they're easy to locate. Use
  constants where possible to indicate that they won't be changed at
  runtime.

## Mock CDN virtual machine

You can develop new tests against a Vagrant VM which uses Varnish to
simulate a CDN. Nginx and stunnel are used to terminate/initiate TLS and
inject headers.
```
               +---------------------------+
         +---> |        Vagrant VM         |-----+
         |     |                           |     |
         | +---| Nginx ¦ Varnish ¦ stunnel | <-+ |
         | |   +---------------------------+   | |
         | |                                   | |
 request-| |-response                          | |
         | |                                   | |
         | |        +-----------------+        | |
         | +------> |     go test     |--------+ |
         |          |                 |          |
         +----------| client ¦ server | <--------+
                    +-----------------+
```

You may need to modify the configuration of the VM in
[`mock_cdn_config/`](/mock_cdn_config) to account for new tests.

To bring up the VM and point the tests at it:
```
vagrant up && vagrant provision
go test -edgeHost 172.16.20.10 -skipVerifyTLS
```

Please note that this is not a complete substitute for the real thing. You
**must** test against a real CDN before submitting any pull requests.
