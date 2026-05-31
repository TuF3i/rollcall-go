package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Auto-CQUPT-Plan/rollcall-go/internal/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type ServiceInfo struct {
	Service    string `json:"service"`
	ClientID   string `json:"client_id"`
	Username   string `json:"edge_username"`
	Addr       string `json:"addr"`
	RegisteredAt string `json:"registered_at"`
}

type Registrar struct {
	client  *clientv3.Client
	leaseID clientv3.LeaseID
	key     string
	closeCh chan struct{}
	done    chan struct{}
}

func New(endpoints, prefix, serviceName, clientID, username, addr string) (*Registrar, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{endpoints},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("etcd 连接失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	lease, err := cli.Grant(ctx, 10)
	if err != nil {
		cli.Close()
		return nil, fmt.Errorf("etcd 租约创建失败: %w", err)
	}

	info := ServiceInfo{
		Service:    serviceName,
		ClientID:   clientID,
		Username:   username,
		Addr:       addr,
		RegisteredAt: time.Now().UTC().Format(time.RFC3339),
	}
	val, _ := json.Marshal(info)

	key := fmt.Sprintf("%s/%s/%s", prefix, serviceName, clientID)
	_, err = cli.Put(ctx, key, string(val), clientv3.WithLease(lease.ID))
	if err != nil {
		cli.Close()
		return nil, fmt.Errorf("etcd 注册失败: %w", err)
	}

	slog.Info(fmt.Sprintf("%s %s %s",
		logger.TagOK("服务已注册"),
		logger.KV("etcd", endpoints),
		logger.KV("key", key)))

	r := &Registrar{
		client:  cli,
		leaseID: lease.ID,
		key:     key,
		closeCh: make(chan struct{}),
		done:    make(chan struct{}),
	}

	go r.keepAlive()

	return r, nil
}

func (r *Registrar) keepAlive() {
	defer close(r.done)

	ch, err := r.client.KeepAlive(context.Background(), r.leaseID)
	if err != nil {
		slog.Error("etcd 租约续期启动失败", "error", err)
		return
	}

	for {
		select {
		case <-r.closeCh:
			return
		case _, ok := <-ch:
			if !ok {
				slog.Warn("etcd 租约续期通道关闭")
				return
			}
		}
	}
}

func (r *Registrar) Close() error {
	close(r.closeCh)
	<-r.done

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := r.client.Revoke(ctx, r.leaseID)
	if err != nil {
		slog.Warn("etcd 租约撤销失败", "error", err)
	}

	r.client.Close()
	slog.Info(fmt.Sprintf("%s %s", logger.TagOK("服务已注销"), logger.KV("key", r.key)))
	return nil
}
