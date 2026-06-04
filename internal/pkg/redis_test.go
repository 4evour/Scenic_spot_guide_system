package pkg

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/scenic-guide/internal/config"
)

func resetRedisState() {
	redisClient = atomic.Pointer[redis.Client]{}
	redisOnce = sync.Once{}
}

func TestGetRedisReturnsNilBeforeInit(t *testing.T) {
	resetRedisState()

	client := GetRedis()
	if client != nil {
		t.Fatalf("GetRedis() = %v, want nil before InitRedis is called", client)
	}
}

func TestInitRedisWithNilConfig(t *testing.T) {
	resetRedisState()

	err := InitRedis(nil)
	if err != nil {
		t.Fatalf("InitRedis(nil) returned error: %v", err)
	}

	if GetRedis() != nil {
		t.Fatalf("GetRedis() should remain nil when InitRedis is called with nil config")
	}
}

func TestInitRedisWithEmptyAddr(t *testing.T) {
	resetRedisState()

	err := InitRedis(&config.RedisConfig{})
	if err != nil {
		t.Fatalf("InitRedis(empty config) returned error: %v", err)
	}

	if GetRedis() != nil {
		t.Fatalf("GetRedis() should remain nil when Redis addr is empty")
	}
}

func TestInitRedisWithInvalidAddr(t *testing.T) {
	resetRedisState()

	err := InitRedis(&config.RedisConfig{Addr: "localhost:63999"})
	if err == nil {
		t.Fatal("InitRedis(invalid addr) should return an error")
	}

	if GetRedis() != nil {
		t.Fatalf("GetRedis() should remain nil when connection fails")
	}
}
