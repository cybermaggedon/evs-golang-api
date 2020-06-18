package evs

import (
	"encoding/binary"
	pb "github.com/cybermaggedon/evs-golang-api/protos"
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
func AddressToIp(addr *pb.Address) net.IP {
	switch a := addr.AddressVariant.(type) {
	case *pb.Address_Ipv4:
		return Int32ToIp(a.Ipv4)
	case *pb.Address_Ipv6:
		return BytesToIp(a.Ipv6)
	case *pb.Address_Port:
		return nil
	default:
		return nil
	}
}

// Converts an Address (compiled protobuf object) to string form.
func AddressToString(addr *pb.Address) string {
	switch a := addr.AddressVariant.(type) {
	case *pb.Address_Ipv4:
		return Int32ToIp(a.Ipv4).String()
	case *pb.Address_Ipv6:
		return BytesToIp(a.Ipv6).String()
	case *pb.Address_Port:
		return strconv.Itoa(int(a.Port))
	default:
		return ""
	}
}
