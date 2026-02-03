package cache

import (
	"bytes"
	"context"
	"testing"
	"time"
)

// TestCacheServiceContract tests the CacheService contract.
func TestCacheServiceContract(t *testing.T) {
	ctx := context.Background()
	svc := NewMockCacheService()

	t.Run("Set_And_Get_Works", func(t *testing.T) {
		key := "test-key"
		value := []byte("test-value")

		err := svc.Set(ctx, key, value, time.Hour)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		result, ok := svc.Get(ctx, key)
		if !ok {
			t.Fatal("expected key to exist")
		}
		if !bytes.Equal(result, value) {
			t.Errorf("expected %s, got %s", value, result)
		}
	})

	t.Run("Get_NonexistentKey_ReturnsFalse", func(t *testing.T) {
		_, ok := svc.Get(ctx, "nonexistent-key")
		if ok {
			t.Error("expected nonexistent key to return false")
		}
	})

	t.Run("Set_OverwritesExisting", func(t *testing.T) {
		key := "overwrite-key"

		err := svc.Set(ctx, key, []byte("value1"), time.Hour)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		err = svc.Set(ctx, key, []byte("value2"), time.Hour)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		result, _ := svc.Get(ctx, key)
		if string(result) != "value2" {
			t.Errorf("expected value2, got %s", result)
		}
	})

	t.Run("TTL_Expiration", func(t *testing.T) {
		key := "expiring-key"

		// Set with very short TTL
		err := svc.Set(ctx, key, []byte("value"), time.Millisecond)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		// Wait for expiration
		time.Sleep(5 * time.Millisecond)

		_, ok := svc.Get(ctx, key)
		if ok {
			t.Error("expected key to be expired")
		}
	})

	t.Run("ZeroTTL_NeverExpires", func(t *testing.T) {
		key := "no-expire-key"

		err := svc.Set(ctx, key, []byte("value"), 0)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		// Should still exist
		_, ok := svc.Get(ctx, key)
		if !ok {
			t.Error("expected key with zero TTL to persist")
		}
	})

	t.Run("Invalidate_ExactKey", func(t *testing.T) {
		key := "invalidate-exact"
		err := svc.Set(ctx, key, []byte("value"), time.Hour)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		err = svc.Invalidate(ctx, key)
		if err != nil {
			t.Fatalf("Invalidate failed: %v", err)
		}

		_, ok := svc.Get(ctx, key)
		if ok {
			t.Error("expected key to be invalidated")
		}
	})

	t.Run("Invalidate_WildcardPattern", func(t *testing.T) {
		// Set multiple keys with same prefix
		if err := svc.Set(ctx, "user:123:profile", []byte("profile"), time.Hour); err != nil {
			t.Fatal(err)
		}
		if err := svc.Set(ctx, "user:123:settings", []byte("settings"), time.Hour); err != nil {
			t.Fatal(err)
		}
		if err := svc.Set(ctx, "user:123:cache", []byte("cache"), time.Hour); err != nil {
			t.Fatal(err)
		}
		if err := svc.Set(ctx, "user:456:profile", []byte("other"), time.Hour); err != nil {
			t.Fatal(err)
		}

		// Invalidate all user:123:* keys
		err := svc.Invalidate(ctx, "user:123:*")
		if err != nil {
			t.Fatalf("Invalidate failed: %v", err)
		}

		// user:123:* should be gone
		if _, ok := svc.Get(ctx, "user:123:profile"); ok {
			t.Error("user:123:profile should be invalidated")
		}
		if _, ok := svc.Get(ctx, "user:123:settings"); ok {
			t.Error("user:123:settings should be invalidated")
		}
		if _, ok := svc.Get(ctx, "user:123:cache"); ok {
			t.Error("user:123:cache should be invalidated")
		}

		// user:456:* should remain
		if _, ok := svc.Get(ctx, "user:456:profile"); !ok {
			t.Error("user:456:profile should still exist")
		}
	})

	t.Run("Invalidate_NonexistentKey_NoError", func(t *testing.T) {
		err := svc.Invalidate(ctx, "nonexistent")
		if err != nil {
			t.Errorf("Invalidate nonexistent key should not error: %v", err)
		}
	})

	t.Run("Set_EmptyValue", func(t *testing.T) {
		key := "empty-value-key"

		err := svc.Set(ctx, key, []byte{}, time.Hour)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		result, ok := svc.Get(ctx, key)
		if !ok {
			t.Error("expected empty value to be stored")
		}
		if len(result) != 0 {
			t.Error("expected empty value")
		}
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		// Test concurrent reads and writes
		done := make(chan bool)

		// Writer goroutine
		go func() {
			for i := 0; i < 100; i++ {
				_ = svc.Set(ctx, "concurrent-key", []byte("value"), time.Hour)
			}
			done <- true
		}()

		// Reader goroutine
		go func() {
			for i := 0; i < 100; i++ {
				svc.Get(ctx, "concurrent-key")
			}
			done <- true
		}()

		// Wait for both
		<-done
		<-done
	})
}
