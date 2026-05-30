package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/fatih/color"
)

var (
	styleTime    = color.New(color.FgHiBlack)
	styleError   = color.New(color.FgRed, color.Bold)
	styleWarn    = color.New(color.FgYellow)
	styleInfo    = color.New(color.FgGreen)
	styleDebug   = color.New(color.FgHiBlack)
	styleAttrKey = color.New(color.FgHiBlack)
)

type PrettyHandler struct {
	w     io.Writer
	level slog.Leveler
	mu    sync.Mutex
	attrs []slog.Attr
	group string
}

func NewPrettyHandler(w io.Writer, level slog.Leveler) *PrettyHandler {
	return &PrettyHandler{w: w, level: level}
}

func (h *PrettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	timeStr := styleTime.Sprint(r.Time.Format("15:04:05"))

	var levelStr string
	switch {
	case r.Level >= slog.LevelError:
		levelStr = styleError.Sprint("ERROR")
	case r.Level >= slog.LevelWarn:
		levelStr = styleWarn.Sprint(" WARN")
	case r.Level >= slog.LevelInfo:
		levelStr = styleInfo.Sprint(" INFO")
	default:
		levelStr = styleDebug.Sprint("DEBUG")
	}

	var attrsStr string
	allAttrs := make([]slog.Attr, 0, len(h.attrs)+int(r.NumAttrs()))
	allAttrs = append(allAttrs, h.attrs...)
	r.Attrs(func(a slog.Attr) bool {
		allAttrs = append(allAttrs, a)
		return true
	})

	for _, a := range allAttrs {
		if a.Key == "component" {
			continue
		}
		attrsStr += fmt.Sprintf("  %s=%s", styleAttrKey.Sprint(a.Key), styleTime.Sprint(a.Value.String()))
	}

	line := fmt.Sprintf("%s %s ▸%s%s\n",
		timeStr,
		levelStr,
		r.Message,
		attrsStr,
	)

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write([]byte(line))
	return err
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs), len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	newAttrs = append(newAttrs, attrs...)
	return &PrettyHandler{w: h.w, level: h.level, attrs: newAttrs, group: h.group}
}

func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	return &PrettyHandler{w: h.w, level: h.level, attrs: h.attrs, group: name}
}

// Setup initializes the global slog logger with the pretty handler.
func Setup(w io.Writer) {
	handler := NewPrettyHandler(w, slog.LevelInfo)
	slog.SetDefault(slog.New(handler))
}

// FormatDuration formats a duration in a human-friendly way.
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.0fm%.0fs", d.Minutes(), float64(d.Nanoseconds()%int64(time.Minute))/float64(time.Second))
}
