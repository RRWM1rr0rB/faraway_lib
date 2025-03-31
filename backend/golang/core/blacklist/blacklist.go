package blacklist

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

type BlockIP struct {
	ipv4 net.IP
	mask *net.IPNet
}

type Blacklist struct {
	sync.RWMutex
	ips    map[string]*BlockIP
	masks  []*BlockIP
	client *redis.Client
}

// Создаём новый Blacklist с Redis
func NewBlacklist(redisAddr string) (*Blacklist, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // если есть пароль, добавь сюда
		DB:       0,
	})

	bl := &Blacklist{
		ips:    make(map[string]*BlockIP),
		client: client,
	}

	// Загружаем данные из Redis в память
	if err := bl.loadFromRedis(); err != nil {
		return nil, err
	}

	return bl, nil
}

// Загружаем чёрный список из Redis в локальный кэш
func (bl *Blacklist) loadFromRedis() error {
	ctx := context.Background()
	keys, err := bl.client.Keys(ctx, "blacklist:*").Result()
	if err != nil {
		return err
	}

	bl.Lock()
	defer bl.Unlock()

	for _, key := range keys {
		ip := strings.TrimPrefix(key, "blacklist:")
		bl.ips[ip] = &BlockIP{ipv4: net.ParseIP(ip)}
	}

	return nil
}

// Добавляем IP в черный список (Redis + Map)
func (bl *Blacklist) AddIP(ip string) error {
	bl.Lock()
	defer bl.Unlock()

	if bl.IsBlacklisted(ip) {
		return nil
	}

	ipv4 := net.ParseIP(ip)
	if ipv4 != nil {
		bl.ips[ipv4.String()] = &BlockIP{ipv4: ipv4}

		// Записываем в Redis с TTL 7 дней
		ctx := context.Background()
		err := bl.client.Set(ctx, "blacklist:"+ipv4.String(), "1", 7*24*time.Hour).Err()
		if err != nil {
			return err
		}

		fmt.Println("blacklist: added IP", ip)
	} else {
		return fmt.Errorf("invalid IP: %s", ip)
	}

	return nil
}

func (bl *Blacklist) IsBlacklisted(ip string) bool {
	bl.RLock()
	defer bl.RUnlock()

	ipv4 := net.ParseIP(ip)
	if ipv4 == nil {
		return false
	}

	if _, ok := bl.ips[ip]; ok {
		return true
	}

	for _, m := range bl.masks {
		if m.mask != nil && m.mask.Contains(ipv4) {
			return true
		}
	}
	return false
}

func (bl *Blacklist) RemoveIP(ip string) error {
	bl.Lock()
	defer bl.Unlock()

	delete(bl.ips, ip)

	// Удаляем из Redis
	ctx := context.Background()
	err := bl.client.Del(ctx, "blacklist:"+ip).Err()
	if err != nil {
		return err
	}

	fmt.Println("blacklist: removed IP", ip)
	return nil
}
