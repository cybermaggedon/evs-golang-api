
package evs

import (
	"encoding/binary"
	"net"
	"strconv"
)

// Converts a 32-bit int to an IP address
func Int32ToIp(ipLong uint32) net.IP {
	ipByte := make([]byte, 4)
	binary.BigEndian.PutUint32(ipByte, ipLong)
	return net.IP(ipByte)
}

// Converts a byte array to an IP address. This is for IPv6 addresses.
func BytesToIp(b []byte) net.IP {
	return net.IP(b)
}

// Converts an Address (compiled protobuf object) to an IP address.
// Return nil if conversion is not possible.
func AddressToIp(addr *Address) net.IP {
	switch a := addr.AddressVariant.(type) {
	case *Address_Ipv4:
		return Int32ToIp(a.Ipv4)
	case *Address_Ipv6:
		return BytesToIp(a.Ipv6)
	case *Address_Port:
		return nil
	default:
		return nil
	}
}

// Converts an Address (compiled protobuf object) to string form.
func AddressToString(addr *Address) string {
	switch a := addr.AddressVariant.(type) {
	case *Address_Ipv4:
		return Int32ToIp(a.Ipv4).String()
	case *Address_Ipv6:
		return BytesToIp(a.Ipv6).String()
	case *Address_Port:
		return strconv.Itoa(int(a.Port))
	default:
		return ""
	}
}
