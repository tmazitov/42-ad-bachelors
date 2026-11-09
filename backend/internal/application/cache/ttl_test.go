package cache

import (
	"testing"
	"time"
)

func TestTTL(t *testing.T) {
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	c := NewTTL[string, int](time.Minute, 2)
	c.now = func() time.Time { return now }

	c.Set("a", 1)
	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatalf("Get(a) = %d, %v", v, ok)
	}
	now = now.Add(time.Minute)
	if _, ok := c.Get("a"); ok {
		t.Error("a should have expired")
	}

	// Full: the expired entry goes first.
	c.Set("b", 2)
	now = now.Add(time.Second) // b is older than c
	c.Set("c", 3)
	if c.Len() != 2 {
		t.Errorf("len = %d, want 2 (expired a evicted)", c.Len())
	}
	// Full again: the oldest live entry (b) goes.
	now = now.Add(time.Second)
	c.Set("d", 4)
	if _, ok := c.Get("b"); ok {
		t.Error("b should have been evicted")
	}
	for _, k := range []string{"c", "d"} {
		if _, ok := c.Get(k); !ok {
			t.Errorf("%s missing", k)
		}
	}
	// Overwriting a key never evicts.
	c.Set("d", 5)
	if v, _ := c.Get("d"); v != 5 || c.Len() != 2 {
		t.Errorf("d = %d, len = %d", v, c.Len())
	}
	c.Delete("d")
	if _, ok := c.Get("d"); ok {
		t.Error("d should be deleted")
	}
}
