class varnish {
  # Varnish 3 for Travis CI.
  if $::lsbdistcodename == "precise" {
    $apt_key = 'C4DEFFEB'

    exec { 'apt-key varnish':
      command => "/usr/bin/apt-key adv --keyserver keyserver.ubuntu.com --recv-keys ${apt_key}",
      unless  => "/usr/bin/apt-key adv --list-keys ${apt_key}",
    } ->
    package { 'apt-transport-https':
      ensure => present,
    } ->
    file { '/etc/apt/sources.list.d/varnish.list':
      content => "deb https://repo.varnish-cache.org/debian/ $::lsbdistcodename varnish-3.0",
    } ~>
    exec { 'update apt':
      command     => '/usr/bin/apt-get update',
      refreshonly => true,
    } ->
    Package['varnish']
  }

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
