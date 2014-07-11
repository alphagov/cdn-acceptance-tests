if !$::varnish_backend_address {
  fail("Facter fact 'varnish_backend_address' is not set")
}

package { 'ssl-cert': }

class { 'nginx':
  require => Package['ssl-cert'],
} ->
class { 'stunnel':
  require => Package['ssl-cert'],
} ->
class { 'varnish': }
