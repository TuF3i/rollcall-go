package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Auto-CQUPT-Plan/rollcall-go/internal/config"
	"github.com/Auto-CQUPT-Plan/rollcall-go/internal/logger"
	"github.com/Auto-CQUPT-Plan/rollcall-go/internal/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type EdgeEntry struct {
	registry.ServiceInfo
}

type Discovery struct {
	client *clientv3.Client
	edges  map[string][]EdgeEntry // key: edge_username
	mu     sync.RWMutex
	log    *slog.Logger
	prefix string
}

func NewDiscovery(endpoints string) (*Discovery, error) {
	eps := strings.Split(endpoints, ",")
	for i := range eps {
		eps[i] = strings.TrimSpace(eps[i])
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   eps,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("etcd 连接失败: %w", err)
	}

	return &Discovery{
		client: cli,
		edges:  make(map[string][]EdgeEntry),
		log:    slog.Default().With("component", "discovery"),
		prefix: fmt.Sprintf("%s/rollcall_edge", config.Cfg.EtcdPrefix),
	}, nil
}

func (d *Discovery) Run(ctx context.Context) {
	go d.watch(ctx)
}

func (d *Discovery) watch(ctx context.Context) {
	d.fullLoad(ctx)

	watchCh := d.client.Watch(ctx, d.prefix, clientv3.WithPrefix())
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.fullLoad(ctx)
		case wresp := <-watchCh:
			for _, ev := range wresp.Events {
				switch ev.Type {
				case clientv3.EventTypePut:
					var info registry.ServiceInfo
					if err := json.Unmarshal(ev.Kv.Value, &info); err != nil {
						continue
					}
					d.addEdge(info)
				case clientv3.EventTypeDelete:
					d.removeEdge(string(ev.Kv.Key))
				}
			}
		}
	}
}

func (d *Discovery) fullLoad(ctx context.Context) {
	resp, err := d.client.Get(ctx, d.prefix, clientv3.WithPrefix())
	if err != nil {
		d.log.Error("etcd 获取失败", "error", err)
		return
	}

	d.mu.Lock()
	d.edges = make(map[string][]EdgeEntry)
	for _, kv := range resp.Kvs {
		var info registry.ServiceInfo
		if err := json.Unmarshal(kv.Value, &info); err != nil {
			continue
		}
		if info.Username == "" {
			continue
		}
		d.edges[info.Username] = append(d.edges[info.Username], EdgeEntry{ServiceInfo: info})
	}
	d.mu.Unlock()

	d.log.Info(fmt.Sprintf("%s %s", logger.Section("Edge 发现"), "在线数: "+fmt.Sprint(len(resp.Kvs))))
}

func (d *Discovery) addEdge(info registry.ServiceInfo) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if info.Username == "" {
		return
	}

	existing := d.edges[info.Username]
	found := false
	for i, e := range existing {
		if e.ClientID == info.ClientID {
			existing[i] = EdgeEntry{ServiceInfo: info}
			found = true
			break
		}
	}
	if !found {
		d.edges[info.Username] = append(existing, EdgeEntry{ServiceInfo: info})
	}

	d.log.Info("Edge 上线", "username", info.Username, "client_id", info.ClientID[:8], "addr", info.Addr)
}

func (d *Discovery) removeEdge(key string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for username, entries := range d.edges {
		newEntries := entries[:0]
		removed := false
		for _, e := range entries {
			expectedKey := fmt.Sprintf("%s/%s", d.prefix, e.ClientID)
			if expectedKey == key {
				removed = true
			} else {
				newEntries = append(newEntries, e)
			}
		}
		if removed {
			if len(newEntries) == 0 {
				delete(d.edges, username)
			} else {
				d.edges[username] = newEntries
			}
			break
		}
	}
}

func (d *Discovery) ListEdges() []EdgeEntry {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var result []EdgeEntry
	for _, entries := range d.edges {
		result = append(result, entries...)
	}
	return result
}

func (d *Discovery) Resolve(username string) (*EdgeEntry, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	entries, ok := d.edges[username]
	if !ok || len(entries) == 0 {
		return nil, false
	}
	return &entries[0], true
}

func (d *Discovery) Close() error {
	return d.client.Close()
}
