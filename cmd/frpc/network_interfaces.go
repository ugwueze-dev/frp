package main

import (
	"net"
)

func getLocalAddresses() ([]string, error) {
	var addresses []string

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			switch v := a.(type) {
			case *net.IPNet:
				ipv4Addr := v.IP.To4()
				if ipv4Addr != nil && !v.IP.IsLoopback() {
					addr := ipv4Addr.String()

					if isInterfaceConnectedToInternet(addr) {
						addresses = append(addresses, addr)
					}
				}
			}
		}
	}

	return addresses, nil
}

func isInterfaceConnectedToInternet(addr string) bool {
	ip, err := net.ResolveTCPAddr("tcp", addr+":0")
	if err != nil {
		return false
	}

	d := net.Dialer{
		LocalAddr: ip,
	}

	for i := 0; i < 3; i++ {
		_, err := d.Dial("tcp", "google.com:80")
		if err == nil {
			return true
		}
	}

	return false
}
