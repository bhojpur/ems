#!/bin/bash
set -e

# a helper script to run tests

if ! which emsd >/dev/null; then
    echo "missing Bhojpur EMS daemon binary" && exit 1
fi

if ! which emslookupd >/dev/null; then
    echo "missing Bhojpur EMS lookup daemon binary" && exit 1
fi

# run emslookupd
LOOKUP_LOGFILE=$(mktemp -t emslookupd.XXXXXXX)
echo "starting Bhojpur EMS lookup daemon"
echo "  logging to $LOOKUP_LOGFILE"
emslookupd >$LOOKUP_LOGFILE 2>&1 &
LOOKUPD_PID=$!

# run emsd configured to use our lookupd above
rm -f *.dat
EMSD_LOGFILE=$(mktemp -t emslookupd.XXXXXXX)
echo "starting Bhojpur EMSd --data-path=/tmp --lookupd-tcp-address=127.0.0.1:4160 --tls-cert=./test/server.pem --tls-key=./test/server.key --tls-root-ca-file=./test/ca.pem"
echo "  logging to $EMSD_LOGFILE"
emsd --data-path=/tmp --lookupd-tcp-address=127.0.0.1:4160 --tls-cert=./test/server.pem --tls-key=./test/server.key --tls-root-ca-file=./test/ca.pem >$EMSD_LOGFILE 2>&1 &
EMSD_PID=$!

sleep 0.3

cleanup() {
    echo "killing Bhojpur EMS daemon PID $EMSD_PID"
    kill -s TERM $EMSD_PID || cat $EMSD_LOGFILE
    echo "killing Bhojpur EMS lookup daemon PID $LOOKUPD_PID"
    kill -s TERM $LOOKUPD_PID || cat $LOOKUP_LOGFILE
}
trap cleanup INT TERM EXIT

go test -v -timeout 60s