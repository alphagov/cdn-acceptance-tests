$ipaddress_vmhost = regsubst($::ipaddress_eth1, '\.\d+$', '.1')

include varnish
include nginx
