#!/usr/bin/env bash

set -eu

sudo FACTER_varnish_backend_address="127.0.0.1" \
  puppet apply --detailed-exitcodes \
  --modulepath mock_cdn_config/modules \
  mock_cdn_config/manifests/site.pp || [ $? -eq 2 ]

go test -edgeHost 127.0.0.1 -skipVerifyTLS -v
