class varnish {
  package { 'varnish':
    ensure => present,
  } ->
  file { '/etc/varnish/default.vcl':
    ensure  => file,
    content => template('varnish/default.vcl.erb'),
  } ~>
  service { 'varnish':
    ensure => running,
  }
}
