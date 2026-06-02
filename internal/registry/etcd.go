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
	Service      string `json:"service"`
	ClientID     string `json:"client_id"`
	Username     string `json:"edge_username"`
	Addr         string `json:"addr"`
	RegisteredAt string `json:"registered_at"`
}

type Registrar struct {
	client  *clientv3.Client
	leaseID clientv3.LeaseID
	key     string
	info    ServiceInfo
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

	r := &Registrar{
		client:  cli,
		key:     fmt.Sprintf("%s/%s/%s", prefix, serviceName, clientID),
		closeCh: make(chan struct{}),
		done:    make(chan struct{}),
	}

	info := ServiceInfo{
		Service:      serviceName,
		ClientID:     clientID,
		Username:     username,
		Addr:         addr,
		RegisteredAt: time.Now().UTC().Format(time.RFC3339),
	}
	r.info = info

	if err := r.register(info); err != nil {
		cli.Close()
		return nil, err
	}

	slog.Info(fmt.Sprintf("%s %s %s",
		logger.TagOK("服务已注册"),
		logger.KV("etcd", endpoints),
		logger.KV("key", r.key)))

	go r.keepAlive()

	return r, nil
}

func (r *Registrar) register(info ServiceInfo) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	lease, err := r.client.Grant(ctx, 10)
	if err != nil {
		return fmt.Errorf("etcd 租约创建失败: %w", err)
	}

	val, _ := json.Marshal(info)
	_, err = r.client.Put(ctx, r.key, string(val), clientv3.WithLease(lease.ID))
	if err != nil {
		return fmt.Errorf("etcd 注册失败: %w", err)
	}

	r.leaseID = lease.ID
	return nil
}

func (r *Registrar) keepAlive() {
	defer close(r.done)

	for {
		ch, err := r.client.KeepAlive(context.Background(), r.leaseID)
		if err != nil {
			slog.Warn("etcd 租约续期失败，5秒后重试", "error", err)
			select {
			case <-r.closeCh:
				return
			case <-time.After(5 * time.Second):
				if err := r.register(r.info); err != nil {
					slog.Warn("etcd 重新注册失败", "error", err)
				}
				continue
			}
		}

		for {
			select {
			case <-r.closeCh:
				return
			case _, ok := <-ch:
				if !ok {
					slog.Warn("etcd 租约续期通道关闭，正在重新注册")
					if err := r.register(r.info); err != nil {
						slog.Warn("etcd 重新注册失败", "error", err)
					}
					goto reconnect
				}
			}
		}

	reconnect:
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
