# Run all tests by default. Set this to a regexp matching
# test names to run if limiting the scope is required.
RUN := '.+'

mock: vagrant-up vagrant-provision go-mock

test: go-test

clean: vagrant-destroy

go-mock:
	go test -edgeHost 172.16.20.10 -skipVerifyTLS -test.v -run $(RUN)

go-test:
	go test -edgeHost $(HOST) -skipVerifyTLS -test.v -run $(RUN)

vagrant-up:
	vagrant up

vagrant-provision:
	vagrant provision

vagrant-destroy:
	vagrant destroy

