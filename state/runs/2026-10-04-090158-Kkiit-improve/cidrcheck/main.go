package main

import (
	"fmt"
	"net"
)

func main() {
	for _, s := range []string{"10.0.0.5/8", "10.0.0.0/8", "192.168.1.0/24", "2001:db8::/32", " 10.0.0.0/8 ", "2001:db8::1/32"} {
		ip, n, err := net.ParseCIDR(s)
		fmt.Printf("in=%q err=%v ip=%v net=%v hostbits=%v\n", s, err, ip, n, err == nil && !ip.Equal(n.IP))
	}
}
