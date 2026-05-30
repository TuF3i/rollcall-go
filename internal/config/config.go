package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/Auto-CQUPT-Plan/rollcall-go/internal/logger"
	"github.com/google/uuid"
)

type Config struct {
	Username             string `json:"username"`
	Password             string `json:"password"`
	CurriculumAPI        string `json:"curriculum_api"`
	CurriculumPreMinutes int    `json:"curriculum_pre_minutes"`
	HTTPPort             *int   `json:"http_port"`
	CenterServerURL      string `json:"center_server_url"`
	CenterServerSecret   string `json:"center_server_secret"`
	AutoLocationCheckin  bool   `json:"auto_location_checkin"`
	AutoNumberCheckin    bool   `json:"auto_number_checkin"`
	PollDelay            int    `json:"poll_delay"`
	TGBotToken           string `json:"tg_bot_token"`
	TGChatID             string `json:"tg_chat_id"`
}

var (
	Cfg      Config
	ClientID string
	DataDir  string

	PauseSharedRollcall atomic.Bool
)

func Load() error {
	DataDir = "data"
	if err := os.MkdirAll(DataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	// Defaults
	defaultPort := 8080
	Cfg = Config{
		CurriculumPreMinutes: 10,
		HTTPPort:             &defaultPort,
		AutoLocationCheckin:  true,
		AutoNumberCheckin:    true,
		PollDelay:            30,
	}

	// Load from file
	cfgPath := filepath.Join(DataDir, "config.json")
	if data, err := os.ReadFile(cfgPath); err == nil {
		if err := json.Unmarshal(data, &Cfg); err != nil {
			slog.Warn("config.json 解析失败，使用默认值", "error", err)
		}
	}

	// Environment variable overrides
	applyEnvOverrides()

	if Cfg.Username == "" || Cfg.Password == "" {
		return fmt.Errorf("username and password are required")
	}

	// Load or generate client ID
	ClientID = loadClientID()

	features := ""
	if Cfg.AutoLocationCheckin {
		features += " 定位"
	}
	if Cfg.AutoNumberCheckin {
		features += " 数字"
	}
	if Cfg.CenterServerURL != "" {
		features += " 共享"
	}
	if features == "" {
		features = " 无"
	}

	slog.Info(fmt.Sprintf("%s %s %s",
		logger.TagOK("配置已加载"),
		logger.KV("客户端", ClientID[:8]+"..."),
		logger.KV("功能", strings.TrimSpace(features))))

	Dump()

	return nil
}

func applyEnvOverrides() {
	if v := os.Getenv("EDGE_USERNAME"); v != "" {
		Cfg.Username = v
	}
	if v := os.Getenv("EDGE_PASSWORD"); v != "" {
		Cfg.Password = v
	}
	if v := os.Getenv("EDGE_CURRICULUM_API"); v != "" {
		Cfg.CurriculumAPI = v
	}
	if v := os.Getenv("EDGE_CURRICULUM_PRE_MINUTES"); v != "" {
		var m int
		if _, err := fmt.Sscanf(v, "%d", &m); err == nil {
			Cfg.CurriculumPreMinutes = m
		}
	}
	if v, ok := os.LookupEnv("EDGE_HTTP_PORT"); ok {
		if v == "" {
			Cfg.HTTPPort = nil
		} else {
			var p int
			if _, err := fmt.Sscanf(v, "%d", &p); err == nil {
				Cfg.HTTPPort = &p
			}
		}
	}
	if v := os.Getenv("EDGE_CENTER_SERVER_URL"); v != "" {
		Cfg.CenterServerURL = v
	}
	if v := os.Getenv("EDGE_CENTER_SERVER_SECRET"); v != "" {
		Cfg.CenterServerSecret = v
	}
	if v := os.Getenv("EDGE_AUTO_LOCATION_CHECKIN"); v != "" {
		lower := strings.ToLower(v)
		Cfg.AutoLocationCheckin = lower == "true" || lower == "1" || lower == "yes"
	}
	if v := os.Getenv("EDGE_AUTO_NUMBER_CHECKIN"); v != "" {
		lower := strings.ToLower(v)
		Cfg.AutoNumberCheckin = lower == "true" || lower == "1" || lower == "yes"
	}
	if v := os.Getenv("POLL_DELAY"); v != "" {
		var m int
		if _, err := fmt.Sscanf(v, "%d", &m); err == nil {
			Cfg.PollDelay = m
		}
	}
	if v := os.Getenv("TG_BOT_TOKEN"); v != "" {
		Cfg.TGBotToken = v
	}
	if v := os.Getenv("TG_CHAT_ID"); v != "" {
		Cfg.TGChatID = v
	}
}

func loadClientID() string {
	// Priority: env var > file > generate
	if v := os.Getenv("EDGE_CLIENT_ID"); v != "" {
		return v
	}

	idPath := filepath.Join(DataDir, "client_id.txt")
	if data, err := os.ReadFile(idPath); err == nil {
		id := strings.TrimSpace(string(data))
		if id != "" {
			return id
		}
	}

	id := uuid.New().String()
	if err := os.WriteFile(idPath, []byte(id), 0o644); err != nil {
		slog.Warn("client_id 保存失败", "error", err)
	}
	return id
}

// Dump prints the full configuration in a structured format.
func Dump() {
	mask := func(s string) string {
		if len(s) > 4 {
			return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
		}
		return strings.Repeat("*", len(s))
	}

	slog.Info(fmt.Sprintf("  %s", logger.Section("账号")))
	slog.Info(fmt.Sprintf("    %s %s", logger.K("用户名"), logger.V(Cfg.Username)))
	if Cfg.Password != "" {
		slog.Info(fmt.Sprintf("    %s %s", logger.K("密码"), logger.Dim(mask(Cfg.Password))))
	}
	slog.Info(fmt.Sprintf("    %s %s", logger.K("客户端ID"), logger.V(ClientID)))

	if Cfg.CenterServerURL != "" {
		slog.Info(fmt.Sprintf("  %s", logger.Section("中心服务器")))
		slog.Info(fmt.Sprintf("    %s %s", logger.K("地址"), logger.V(Cfg.CenterServerURL)))
		if Cfg.CenterServerSecret != "" {
			slog.Info(fmt.Sprintf("    %s %s", logger.K("密钥"), logger.Dim(mask(Cfg.CenterServerSecret))))
		}
	}

	slog.Info(fmt.Sprintf("  %s", logger.Section("签到")))
	slog.Info(fmt.Sprintf("    %s %s", logger.K("轮询间隔"), logger.V(fmt.Sprintf("%ds", Cfg.PollDelay))))
	slog.Info(fmt.Sprintf("    %s %s", logger.K("自动定位"), logger.V(boolStr(Cfg.AutoLocationCheckin))))
	slog.Info(fmt.Sprintf("    %s %s", logger.K("自动数字"), logger.V(boolStr(Cfg.AutoNumberCheckin))))

	if Cfg.CurriculumAPI != "" {
		slog.Info(fmt.Sprintf("  %s", logger.Section("课表")))
		slog.Info(fmt.Sprintf("    %s %s", logger.K("API"), logger.V(Cfg.CurriculumAPI)))
		slog.Info(fmt.Sprintf("    %s %s", logger.K("提前时间"), logger.V(fmt.Sprintf("%dmin", Cfg.CurriculumPreMinutes))))
	}

	if Cfg.TGBotToken != "" && Cfg.TGChatID != "" {
		slog.Info(fmt.Sprintf("  %s", logger.Section("通知")))
		slog.Info(fmt.Sprintf("    %s %s", logger.K("机器人"), logger.V(mask(Cfg.TGBotToken))))
		slog.Info(fmt.Sprintf("    %s %s", logger.K("聊天ID"), logger.V(Cfg.TGChatID)))
	}

	if Cfg.HTTPPort != nil {
		slog.Info(fmt.Sprintf("  %s", logger.Section("HTTP")))
		slog.Info(fmt.Sprintf("    %s %s", logger.K("端口"), logger.V(fmt.Sprintf(":%d", *Cfg.HTTPPort))))
	}
}

func boolStr(b bool) string {
	if b {
		return "✔"
	}
	return "✘"
}

func CookiesPath() string {
	return filepath.Join(DataDir, "cookies.json")
}

func CurriculumCachePath() string {
	return filepath.Join(DataDir, "curriculum_cache.json")
}
