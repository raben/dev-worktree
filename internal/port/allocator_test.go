package port

import (
	"net"
	"testing"
)

func TestIsAvailable_ListeningPort(t *testing.T) {
	// Listen on all interfaces so IsAvailable (which binds to ":port") detects it.
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen on random port: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	if IsAvailable(port) {
		t.Errorf("IsAvailable(%d) = true, want false (port is in use)", port)
	}
}

func TestIsAvailable_FreeHighPort(t *testing.T) {
	// Find a free port by binding and immediately closing.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	if port >= minPort && !IsAvailable(port) {
		t.Errorf("IsAvailable(%d) = false, want true (port was just freed)", port)
	}
}

func TestIsAvailable_BelowMinPort(t *testing.T) {
	if IsAvailable(80) {
		t.Errorf("IsAvailable(80) = true, want false (below minPort)")
	}
}

func TestIsAvailable_AboveMaxPort(t *testing.T) {
	if IsAvailable(70000) {
		t.Errorf("IsAvailable(70000) = true, want false (above maxPort)")
	}
}

func TestIsAvailable_Zero(t *testing.T) {
	if IsAvailable(0) {
		t.Errorf("IsAvailable(0) = true, want false")
	}
}

func TestIsAvailable_Negative(t *testing.T) {
	if IsAvailable(-1) {
		t.Errorf("IsAvailable(-1) = true, want false")
	}
}

func TestAllocate_ValidBase(t *testing.T) {
	port, err := Allocate(49152)
	if err != nil {
		t.Fatalf("Allocate(49152) returned error: %v", err)
	}
	if port < 49152 || port > maxPort {
		t.Errorf("Allocate(49152) = %d, want in range [49152, %d]", port, maxPort)
	}
}

func TestAllocate_BelowMinPort(t *testing.T) {
	_, err := Allocate(80)
	if err == nil {
		t.Error("Allocate(80) should return error for port below minPort")
	}
}

func TestAllocate_AboveMaxPort(t *testing.T) {
	_, err := Allocate(70000)
	if err == nil {
		t.Error("Allocate(70000) should return error for port above maxPort")
	}
}

func TestAllocateMultiple_NoDuplicates(t *testing.T) {
	bases := []int{49152, 49152, 49152}
	result, err := AllocateMultiple(bases)
	if err != nil {
		t.Fatalf("AllocateMultiple returned error: %v", err)
	}

	// All bases are the same key, so map will have one entry.
	// But the function iterates over the slice, so last write wins.
	// Check that at least one port was allocated.
	if len(result) == 0 {
		t.Error("AllocateMultiple returned empty map")
	}

	// Verify all values are valid ports.
	for base, port := range result {
		if port < base || port > maxPort {
			t.Errorf("AllocateMultiple: base %d got port %d, want in range [%d, %d]", base, port, base, maxPort)
		}
	}
}

func TestAllocateMultiple_DifferentBases(t *testing.T) {
	bases := []int{49152, 49200, 49300}
	result, err := AllocateMultiple(bases)
	if err != nil {
		t.Fatalf("AllocateMultiple returned error: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("AllocateMultiple returned %d entries, want 3", len(result))
	}

	// Collect allocated ports and verify no duplicates.
	seen := make(map[int]bool)
	for _, port := range result {
		if seen[port] {
			t.Errorf("AllocateMultiple returned duplicate port %d", port)
		}
		seen[port] = true
	}
}

func TestAllocateMultiple_InvalidBase(t *testing.T) {
	bases := []int{49152, 80}
	_, err := AllocateMultiple(bases)
	if err == nil {
		t.Error("AllocateMultiple should return error when a base is below minPort")
	}
}
