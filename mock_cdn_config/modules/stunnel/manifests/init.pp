class stunnel {
  package { 'stunnel':
    ensure => present,
  } ->
  file { '/etc/default/stunnel4':
    ensure  => file,
    content => template('stunnel/stunnel4.erb'),
  } ->
  file { '/etc/stunnel/cdn.conf':
    ensure => file,
    content => template('stunnel/cdn.conf.erb'),
  } ~>
  service { 'stunnel4':
    ensure => running,
  }
}
