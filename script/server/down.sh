#!/bin/bash

# $1 device name
# $2 ip
# $3 netmask

ip route del $3 dev $1
ip link set dev $1 down
ip addr del $2 dev $1
