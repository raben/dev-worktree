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

// Allocation represents a port that has been allocated and is being held open.
type Allocation struct {
	Port     int
	listener net.Listener
}

// Release releases the held port so Docker can bind it.
func (a *Allocation) Release() {
	if a.listener != nil {
		a.listener.Close()
	}
}

// IsAvailable checks if a TCP port is available for listening.
// Uses net.Listen to actually try binding (more reliable than lsof/ss).
// Note: This has a TOCTOU race; prefer AllocateAndHold for production use.
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
// Note: This has a TOCTOU race; prefer AllocateAndHold for production use.
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
// Note: This has a TOCTOU race; prefer AllocateMultipleAndHold for production use.
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

// AllocateAndHold finds an available port and keeps it held open via a listener.
// The caller must call Release() on the returned Allocation before Docker binds the port.
func AllocateAndHold(base int) (*Allocation, error) {
	if base < minPort || base > maxPort {
		return nil, fmt.Errorf("port %d: base port out of valid range (%d-%d)", base, minPort, maxPort)
	}
	for p := base; p <= maxPort; p++ {
		ln, err := net.Listen("tcp", ":"+strconv.Itoa(p))
		if err != nil {
			continue
		}
		return &Allocation{Port: p, listener: ln}, nil
	}
	return nil, fmt.Errorf("port %d: all ports exhausted (tried %d-%d)", base, base, maxPort)
}

// AllocateMultipleAndHold allocates multiple ports, keeping them all held open.
// The caller must call Release() on each Allocation before Docker binds the ports.
// On error, all already-held ports are released before returning.
func AllocateMultipleAndHold(bases []int) ([]*Allocation, error) {
	var allocs []*Allocation
	used := make(map[int]bool)

	for _, base := range bases {
		if base < minPort || base > maxPort {
			for _, a := range allocs {
				a.Release()
			}
			return nil, fmt.Errorf("port %d: base port out of valid range (%d-%d)", base, minPort, maxPort)
		}
		allocated := false
		for p := base; p <= maxPort; p++ {
			if used[p] {
				continue
			}
			ln, err := net.Listen("tcp", ":"+strconv.Itoa(p))
			if err != nil {
				continue
			}
			allocs = append(allocs, &Allocation{Port: p, listener: ln})
			used[p] = true
			allocated = true
			break
		}
		if !allocated {
			for _, a := range allocs {
				a.Release()
			}
			return nil, fmt.Errorf("port %d: all ports exhausted (tried %d-%d)", base, base, maxPort)
		}
	}
	return allocs, nil
}
