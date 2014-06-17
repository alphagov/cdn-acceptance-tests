class varnish {
  package { 'varnish':
    ensure => present,
  } ->
  file { '/etc/varnish/default.vcl':
    ensure  => file,
    content => template('varnish/default.vcl.erb'),
    notify  => Exec['varnish-restart'],
  } ->
  service { 'varnish':
    ensure => running,
  }

  # `service varnish restart` doesn't return the right exit code!
  exec { 'varnish-restart':
    command     => '/usr/sbin/service varnish stop; /usr/sbin/service varnish start',
    refreshonly => true,
  }
}
