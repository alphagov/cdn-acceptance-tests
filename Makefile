# Run all tests by default. Set this to a regexp matching
# test names to run if limiting the scope is required.
RUN := '.+'

mock: vagrant-up vagrant-provision go-mock

travis: puppet-travis go-travis

test: go-test

clean: vagrant-destroy

go-mock:
	go test -edgeHost 172.16.20.10 -skipVerifyTLS -test.v -run $(RUN)

go-travis:
	go test -edgeHost 127.0.0.1 -skipVerifyTLS -test.v -run $(RUN)

go-test:
	go test -edgeHost $(HOST) -skipVerifyTLS -test.v -run $(RUN)

puppet-travis:
	sudo FACTER_varnish_backend_address="127.0.0.1" \
		puppet apply --detailed-exitcodes \
		--modulepath mock_cdn_config/modules mock_cdn_config/manifests/site.pp \
		|| [ $$? -eq 2 ]

vagrant-up:
	vagrant up

vagrant-provision:
	vagrant provision

vagrant-destroy:
	vagrant destroy

