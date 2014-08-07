#!/bin/sh
test -z "$TEST_EDGEHOST" && export TEST_EDGEHOST="172.16.20.10"
if [ "$TEST_VERIFYTLS" = "yes" ]; then
    export TEST_ARGS="";
else
    export TEST_ARGS="-skipVerifyTLS";
fi
if [ "$TEST_FAILOVER" = "no" ]; then
    export TEST_ARGS="${TEST_ARGS} -skipFailover"
fi
echo "Running: go test -edgeHost $TEST_EDGEHOST $TEST_ARGS"
go test -edgeHost $TEST_EDGEHOST $TEST_ARGS
