#!/bin/bash

# $1 device name
# $2 ip
# $3 netmask

ip addr add $2 dev $1
ip link set dev $1 up
ip route add $3 dev $1
