# Ears
A utility to scan a given subnet for responsive IP addresses.  Currently
only supports ICMP pinging.  Future releases will support TCP pinging
as well.

## Build

To install, run

    go get -u github.com/sh3rp/ears

## Usage
Run

    ears -i eth0

to scan the IP network configured on eth0.