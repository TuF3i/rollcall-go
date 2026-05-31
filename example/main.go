package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/net/context"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type ServiceInfo struct {
	Service      string `json:"service"`
	ClientID     string `json:"client_id"`
	Username     string `json:"edge_username"`
	Addr         string `json:"addr"`
	RegisteredAt string `json:"registered_at"`
}

type Rollcall struct {
	RollcallID   int    `json:"rollcall_id"`
	Source       string `json:"source"`
	Status       string `json:"status"`
	CourseTitle  string `json:"course_title"`
	RollcallTime string `json:"rollcall_time"`
}

type HealthResponse struct {
	Status   string `json:"status"`
	ClientID string `json:"client_id"`
}

func main() {
	etcdEndpoints := flag.String("etcd", "127.0.0.1:2379", "etcd 地址，多个用逗号分隔")
	prefix := flag.String("prefix", "/rollcall", "etcd 键前缀")
	username := flag.String("username", "", "要查询的 edge_username（必填）")
	flag.Parse()

	if *username == "" {
		fmt.Println("请指定 --username 参数")
		fmt.Println("")
		fmt.Println("使用示例:")
		fmt.Println("  go run main.go --etcd 127.0.0.1:2379 --username 1697666")
		return
	}

	// 1. 从 etcd 发现服务
	services, err := discover(*etcdEndpoints, *prefix, *username)
	if err != nil {
		fmt.Printf("查询失败: %v\n", err)
		return
	}

	fmt.Printf("找到 %d 个匹配 %s 的节点:\n", len(services), *username)
	for _, svc := range services {
		fmt.Printf("  %s\n", svc.Addr)
	}

	target := services[0]

	// 2. 根据协议模式调用 API
	if strings.HasPrefix(target.Addr, "rpc://") {
		callRPC(target)
	} else {
		callHTTP(target)
	}
}

// ============================================================
// HTTP 模式
// ============================================================

func callHTTP(svc discoveredService) {
	base := svc.Addr
	if !strings.HasPrefix(base, "http://") {
		base = "http://" + base
	}

	fmt.Printf("\n--- HTTP API 调用 ---\n")

	body, err := httpGet(base + "/health")
	if err != nil {
		fmt.Printf("health 请求失败: %v\n", err)
		return
	}
	var health HealthResponse
	json.Unmarshal([]byte(body), &health)
	fmt.Printf("[health] status=%s  client_id=%s\n", health.Status, health.ClientID)

	body, err = httpGet(base + "/rollcalls")
	if err != nil {
		fmt.Printf("rollcalls 请求失败: %v\n", err)
		return
	}
	var rollcalls []Rollcall
	json.Unmarshal([]byte(body), &rollcalls)
	fmt.Printf("[rollcalls] 当前活跃签到: %d 个\n", len(rollcalls))
	for _, r := range rollcalls {
		fmt.Printf("  · [%s] %s — %s (ID: %d)\n", r.Source, r.CourseTitle, r.Status, r.RollcallID)
	}

	body, err = httpGet(base + "/pause_shared")
	if err != nil {
		fmt.Printf("pause_shared 请求失败: %v\n", err)
		return
	}
	fmt.Printf("[pause_shared] %s\n", body)
}

func httpGet(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}

// ============================================================
// RPC 模式（提示信息）
// ============================================================

func callRPC(svc discoveredService) {
	addr := strings.TrimPrefix(svc.Addr, "rpc://")

	fmt.Printf("\n--- RPC API 调用 ---\n")
	fmt.Printf("该实例以 RPC 模式运行，地址为 %s\n", addr)
	fmt.Printf("需使用 Kitex 客户端通过 Thrift 协议调用\n")
	fmt.Printf("接口定义: api/kitex/edge.thrift\n")
	fmt.Printf("\nimport \"github.com/Auto-CQUPT-Plan/rollcall-go/internal/rpc/kitex_gen/edge/edgeservice\"\n")
	fmt.Printf("\nclients, _ := edgeservice.NewClient(\"rollcall_edge\", client.WithHostPorts(%q))\n", addr)
	fmt.Printf("health, _ := clients.Health(context.Background())\n")
	fmt.Printf("rollcalls, _ := clients.GetRollcalls(context.Background())\n")
}

// ============================================================
// Etcd 服务发现
// ============================================================

type discoveredService struct {
	Addr     string
	Username string
	ClientID string
}

func discover(endpoints, prefix, username string) ([]discoveredService, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: strings.Split(endpoints, ","),
		Context:   context.Background(),
	})
	if err != nil {
		return nil, fmt.Errorf("etcd 连接失败: %w", err)
	}
	defer cli.Close()

	key := fmt.Sprintf("%s/rollcall_edge/", prefix)
	resp, err := cli.Get(context.Background(), key, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("etcd 查询失败: %w", err)
	}

	var result []discoveredService
	for _, kv := range resp.Kvs {
		var info ServiceInfo
		if err := json.Unmarshal(kv.Value, &info); err != nil {
			continue
		}
		if info.Username != username {
			continue
		}
		addr := info.Addr
		if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "rpc://") {
			addr = "http://" + addr
		}
		result = append(result, discoveredService{
			Addr:     addr,
			Username: info.Username,
			ClientID: info.ClientID,
		})
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("未找到用户 %s 的注册节点", username)
	}
	return result, nil
}
