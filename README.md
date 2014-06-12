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

When testing a real CDN, the tests must be run on a server that the CDN can
connect to.

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

## Writing tests

When writing new tests please be sure to:

- group the test in a file with other tests of similar behaviour e.g.
  "custom failover"
- use a consistent naming prefix for the functions that so that they can be
  run as a group e.g. `func TestCustomFailover…(…)`
- always call `SwitchHandler(…)` at the beginning of a test for any
  `CDNServeMux` you intend to use during that test. Even if it is just to
  reset the handler to an empty function. Otherwise you may experience
  undesired effects such as runtime panics.
- Define static inputs such as "number of requests" or "time between
  requests" at the beginning of the test so that they're easy to locate. Use
  constants where possible to indicate that they won't be changed at
  runtime.

## Mock CDN virtual machine

You can use the included Vagrant VM, which runs Nginx and Varnish, to mock
CDN behaviour. This can be useful when developing new tests or
functionality when working offline or in parallel to someone else.

It is unlikely that *all* tests will run successfully. If you want a
particular test to pass you may need to modify the Nginx or Varnish configs
in [`mock_cdn_config/`](/mock_cdn_config) accordingly.

To bring up the VM and point the tests at it:
```
vagrant up && vagrant provision
go test -edgeHost 172.16.20.10 -skipVerifyTLS
```

Please note that this is not a complete substitute for the real thing. You
**must** test against a real CDN before submitting any pull requests.
