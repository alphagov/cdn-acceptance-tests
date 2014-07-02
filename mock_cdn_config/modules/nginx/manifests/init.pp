class nginx {
  package { 'nginx':
    ensure => present,
  } ->
  package { 'ssl-cert':
    ensure => present,
  } ->
  file { '/etc/nginx/sites-enabled/default':
    ensure  => file,
    content => template('nginx/default.erb'),
  } ~>
  service { 'nginx':
    ensure => running,
  }
}
