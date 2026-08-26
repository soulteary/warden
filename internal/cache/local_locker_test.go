package cache

import (
	"testing"
)

// TestLocalLocker_NewAndLockUnlock covers LocalLocker constructor and basic Lock/Unlock.
// Additional LocalLocker cases (re-entrancy, concurrent, multiple keys, unlock non-existent)
// are in cache_test.go (TestLocalLocker_Basic, TestLocalLocker_Concurrent, etc.).
func TestLocalLocker_NewAndLockUnlock(t *testing.T) {
	locker := NewLocalLocker()
	if locker == nil {
		t.Fatal("NewLocalLocker() must not return nil")
	}

	key := "local-locker-test-key"
	ok, err := locker.Lock(key)
	if err != nil {
		t.Fatalf("Lock(%q): %v", key, err)
	}
	if !ok {
		t.Errorf("Lock(%q) = false, want true", key)
	}

	err = locker.Unlock(key)
	if err != nil {
		t.Fatalf("Unlock(%q): %v", key, err)
	}
}
