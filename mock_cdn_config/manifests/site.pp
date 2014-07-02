if !$::varnish_backend_address {
  fail("Facter fact 'varnish_backend_address' is not set")
}

include varnish
include nginx
