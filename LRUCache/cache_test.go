package lru

import (
	"testing"
	"time"
)

func TestCacheGetReturnsStoredValue(t *testing.T) {
	cache := New(2, time.Minute)
	cache.Set("language", "Go")
	got, ok := cache.Get("language")
	if !ok {
		t.Fatalf("expected key to be present")
	}
	if got != "Go" {
		t.Fatalf("got %v, want %v", got, "Go")
	}
}

func TestCacheEvictsLeastRecentlyUsedValue(t *testing.T) {
	cache := New(2, time.Minute)

	cache.Set("a", "A")
	cache.Set("b", "B")
	cache.Set("c", "C")

	if _, ok := cache.Get("a"); ok {
		t.Fatal("Get returned evicted key a")
	}

	if _, ok := cache.Get("b"); !ok {
		t.Fatal("Get did not return key b")
	}
	if _, ok := cache.Get("c"); !ok {
		t.Fatal("Get did not return key c")
	}
}

func TestCacheGetRefreshesRecency(t *testing.T) {
	cache := New(2, time.Minute)

	cache.Set("a", "A")
	cache.Set("b", "B")
	cache.Get("a")
	cache.Set("c", "C")

	if _, ok := cache.Get("b"); ok {
		t.Fatal("Get returned evicted key b")
	}
}

func TestCacheExpiresValues(t *testing.T) {
	now := time.Now()
	cache := New(2, time.Minute)
	cache.now = func() time.Time {
		return now
	}

	cache.Set("a", "A")
	now = now.Add(2 * time.Minute)

	if _, ok := cache.Get("a"); ok {
		t.Fatal("Get returned expired value")
	}
}

func TestCacheZeroTTLDoesNotExpireValues(t *testing.T) {
	cache := New(2, 0)

	cache.Set("a", "A")

	if got, ok := cache.Get("a"); !ok || got != "A" {
		t.Fatalf("Get returned (%q, %v), want (%q, true)", got, ok, "A")
	}
}

func TestCacheMinimumCapacityIsOne(t *testing.T) {
	cache := New(0, time.Minute)

	cache.Set("a", "A")
	cache.Set("b", "B")

	if _, ok := cache.Get("a"); ok {
		t.Fatal("cache retained a with capacity one")
	}
	if _, ok := cache.Get("b"); !ok {
		t.Fatal("cache evicted b with capacity one")
	}
}
