package port

import (
	"fmt"
	"net"
	"strconv"
)

const (
	minPort = 1024
	maxPort = 65535
)

// IsAvailable checks if a TCP port is available for listening.
// Uses net.Listen to actually try binding (more reliable than lsof/ss).
func IsAvailable(port int) bool {
	if port < minPort || port > maxPort {
		return false
	}
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// Allocate finds an available port starting from the base port.
// Tries base, base+1, base+2, ... up to 65535.
// Returns the allocated port number or an error if no port is available.
func Allocate(base int) (int, error) {
	if base < minPort || base > maxPort {
		return 0, fmt.Errorf("port %d: base port out of valid range (%d-%d)", base, minPort, maxPort)
	}
	for p := base; p <= maxPort; p++ {
		if IsAvailable(p) {
			return p, nil
		}
	}
	return 0, fmt.Errorf("port %d: all ports exhausted (tried %d-%d)", base, base, maxPort)
}

// AllocateMultiple allocates multiple ports from a list of base ports.
// Returns a map of base_port -> allocated_port.
// Each allocation is independent (doesn't skip ports allocated for others in same call).
// But it DOES track already-allocated ports within the same call to avoid duplicates.
func AllocateMultiple(bases []int) (map[int]int, error) {
	result := make(map[int]int, len(bases))
	used := make(map[int]bool)

	for _, base := range bases {
		if base < minPort || base > maxPort {
			return nil, fmt.Errorf("port %d: base port out of valid range (%d-%d)", base, minPort, maxPort)
		}
		allocated := false
		for p := base; p <= maxPort; p++ {
			if used[p] {
				continue
			}
			if IsAvailable(p) {
				result[base] = p
				used[p] = true
				allocated = true
				break
			}
		}
		if !allocated {
			return nil, fmt.Errorf("port %d: all ports exhausted (tried %d-%d)", base, base, maxPort)
		}
	}
	return result, nil
}
