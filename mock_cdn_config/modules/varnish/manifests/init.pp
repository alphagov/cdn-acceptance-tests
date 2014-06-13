class varnish {
  package { 'varnish':
    ensure => present,
  } ->
  file { '/etc/varnish/default.vcl':
    ensure  => file,
    content => template('varnish/default.vcl.erb'),
    notify  => Exec['varnish-reload'],
  } ->
  service { 'varnish':
    ensure => running,
  }

  # `service varnish reload` doesn't return the right exit code!
  exec { 'varnish-reload':
    command     => '/usr/share/varnish/reload-vcl',
    logoutput   => true,
    refreshonly => true,
  }
}
