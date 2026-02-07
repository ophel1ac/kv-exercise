package kv

import (
	"errors"
	"fmt"
	"testing"
	"testing/synctest"
)

func TestConcurrentWithSyncTest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		db := New[string, int]()

		for i := 0; i < 10; i++ {
			go func(n int) {
				key := fmt.Sprintf("key%d", n)
				db.Set(key, n)
			}(i)
		}

		synctest.Wait()

		for i := 0; i < 10; i++ {
			key := fmt.Sprintf("key%d", i)
			val, err := db.Get(key)
			if err != nil {
				t.Errorf("Expected key %s to exist, got error: %v", key, err)
			}
			if val != i {
				t.Errorf("Expected value %d for key %s, got %d", i, key, val)
			}
		}
	})
}

func TestConcurrentSetSameKeyWithSyncTest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		db := New[string, int]()
		key := "shared-key"

		for i := 0; i < 50; i++ {
			go func(n int) {
				db.Set(key, n)
			}(i)
		}

		synctest.Wait()

		val, err := db.Get(key)
		if err != nil {
			t.Errorf("Expected key %s to exist, got error: %v", key, err)
		}
		if val < 0 || val >= 50 {
			t.Errorf("Expected value between 0 and 49, got %d", val)
		}
	})
}

func TestConcurrentGetAndSetWithSyncTest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		db := New[string, string]()

		// Initialize some keys
		for i := 0; i < 5; i++ {
			db.Set(fmt.Sprintf("key%d", i), fmt.Sprintf("initial-%d", i))
		}

		// Launch concurrent readers and writers
		for i := 0; i < 20; i++ {
			// Reader
			go func(n int) {
				key := fmt.Sprintf("key%d", n%5)
				val, err := db.Get(key)
				if err != nil {
					t.Errorf("Unexpected error reading key %s: %v", key, err)
				} else {
					t.Logf("Read key %s: %s", key, val)
				}
			}(i)

			// Writer
			go func(n int) {
				key := fmt.Sprintf("key%d", n%5)
				db.Set(key, fmt.Sprintf("updated-%d", n))
			}(i)
		}

		// Wait for all operations to complete
		synctest.Wait()

		// Verify all keys still exist
		for i := 0; i < 5; i++ {
			key := fmt.Sprintf("key%d", i)
			_, err := db.Get(key)
			if err != nil {
				t.Errorf("Expected key %s to exist after concurrent operations", key)
			}
		}
	})
}

func TestConcurrentDeleteWithSyncTest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		db := New[string, int]()

		// Initialize keys
		for i := 0; i < 10; i++ {
			db.Set(fmt.Sprintf("key%d", i), i)
		}

		// Launch concurrent deletes
		for i := 0; i < 10; i++ {
			go func(n int) {
				key := fmt.Sprintf("key%d", n)
				db.Delete(key)
			}(i)
		}

		// Wait for all deletes to complete
		synctest.Wait()

		// Verify all keys are deleted
		for i := 0; i < 10; i++ {
			key := fmt.Sprintf("key%d", i)
			_, err := db.Get(key)
			if err == nil {
				t.Errorf("Expected key %s to be deleted", key)
			}
			if !errors.Is(err, &ErrNotFound[string]{}) {
				t.Errorf("Expected ErrNotFound for key %s, got %v", key, err)
			}
		}
	})
}

func TestConcurrentMixedOperationsWithSyncTest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		db := New[string, int]()
		key := "test-key"

		// Initialize key
		db.Set(key, 100)

		// Launch mixed operations
		for i := 0; i < 30; i++ {
			go func(n int) {
				switch n % 3 {
				case 0:
					db.Set(key, n)
				case 1:
					val, err := db.Get(key)
					if err == nil {
						t.Logf("Read value: %d", val)
					}
				case 2:
					db.Delete(key)
				}
			}(i)
		}

		// Wait for all operations to complete
		synctest.Wait()

		// Test passes if no race conditions or panics occur
		t.Log("All mixed operations completed without race conditions")
	})
}

// BenchmarkConcurrentSet benchmarks concurrent writes
func BenchmarkConcurrentSet(b *testing.B) {
	db := New[int, string]()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			db.Set(i, fmt.Sprintf("value-%d", i))
			i++
		}
	})
}

// BenchmarkConcurrentGet benchmarks concurrent reads
func BenchmarkConcurrentGet(b *testing.B) {
	db := New[int, string]()
	numKeys := 1000

	// Initialize keys
	for i := 0; i < numKeys; i++ {
		db.Set(i, fmt.Sprintf("value-%d", i))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			db.Get(i % numKeys)
			i++
		}
	})
}

// BenchmarkConcurrentMixed benchmarks mixed read/write operations
func BenchmarkConcurrentMixed(b *testing.B) {
	db := New[int, string]()
	numKeys := 1000

	// Initialize keys
	for i := 0; i < numKeys; i++ {
		db.Set(i, fmt.Sprintf("value-%d", i))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := i % numKeys
			if i%2 == 0 {
				db.Get(key)
			} else {
				db.Set(key, fmt.Sprintf("value-%d", i))
			}
			i++
		}
	})
}
