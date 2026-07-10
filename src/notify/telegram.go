package notify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxTelegramMessageBytes = 4000

type Config struct {
	Token       string
	OwnerChatID int64
	APIEndpoint string
	Timeout     time.Duration
	DedupWindow time.Duration
	MaxStack    int
}

type Telegram struct {
	config Config
	client *http.Client

	mu          sync.Mutex
	sentEvents  map[string]struct{}
	panicEvents map[string]panicEvent
}

type panicEvent struct {
	lastSent   time.Time
	suppressed int
}

type apiResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

func New(config Config) *Telegram {
	if config.Timeout <= 0 {
		config.Timeout = 8 * time.Second
	}
	if config.DedupWindow <= 0 {
		config.DedupWindow = 5 * time.Minute
	}
	if config.MaxStack <= 0 {
		config.MaxStack = 8192
	}
	return &Telegram{
		config:      config,
		client:      &http.Client{Timeout: config.Timeout},
		sentEvents:  make(map[string]struct{}),
		panicEvents: make(map[string]panicEvent),
	}
}

func NewWithClient(config Config, client *http.Client) *Telegram {
	n := New(config)
	if client != nil {
		n.client = client
	}
	return n
}

func (n *Telegram) Enabled() bool {
	return n != nil && n.config.OwnerChatID != 0 && n.config.Token != ""
}

func (n *Telegram) Send(ctx context.Context, text string) error {
	if !n.Enabled() {
		return nil
	}
	text = truncateUTF8(text, maxTelegramMessageBytes)
	values := url.Values{}
	values.Set("chat_id", strconv.FormatInt(n.config.OwnerChatID, 10))
	values.Set("text", text)
	values.Set("disable_web_page_preview", "true")

	endpoint := fmt.Sprintf(n.config.APIEndpoint, n.config.Token, "sendMessage")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return fmt.Errorf("创建 Telegram 通知请求: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送 Telegram 通知失败: %T", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return fmt.Errorf("读取 Telegram 通知响应: %w", err)
	}
	var result apiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析 Telegram 通知响应: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || !result.OK {
		return fmt.Errorf("Telegram 通知失败: status=%d description=%s", resp.StatusCode, truncateUTF8(result.Description, 256))
	}
	return nil
}

func (n *Telegram) SendEvent(ctx context.Context, key, text string) error {
	if !n.Enabled() {
		return nil
	}
	n.mu.Lock()
	if _, exists := n.sentEvents[key]; exists {
		n.mu.Unlock()
		return nil
	}
	n.sentEvents[key] = struct{}{}
	n.mu.Unlock()

	if err := n.Send(ctx, text); err != nil {
		n.mu.Lock()
		delete(n.sentEvents, key)
		n.mu.Unlock()
		return err
	}
	return nil
}

func (n *Telegram) SendPanic(ctx context.Context, component string, recovered any, stack []byte, metadata string) error {
	if !n.Enabled() {
		return nil
	}
	panicText := truncateUTF8(fmt.Sprint(recovered), 512)
	stack = truncateBytes(stack, n.config.MaxStack)
	fingerprint := panicFingerprint(component, panicText, stack)
	now := time.Now()

	n.mu.Lock()
	event := n.panicEvents[fingerprint]
	if !event.lastSent.IsZero() && now.Sub(event.lastSent) < n.config.DedupWindow {
		event.suppressed++
		n.panicEvents[fingerprint] = event
		n.mu.Unlock()
		return nil
	}
	suppressed := event.suppressed
	n.panicEvents[fingerprint] = panicEvent{lastSent: now}
	n.mu.Unlock()

	message := fmt.Sprintf("🚨 任务发生 panic\n组件: %s\n摘要: %s\n事件: %s", component, panicText, fingerprint)
	if metadata != "" {
		message += "\n" + truncateUTF8(metadata, 512)
	}
	if suppressed > 0 {
		message += fmt.Sprintf("\n已抑制重复事件: %d", suppressed)
	}
	if len(stack) > 0 {
		message += "\n\nStack:\n" + string(stack)
	}
	if err := n.Send(ctx, message); err != nil {
		n.mu.Lock()
		current := n.panicEvents[fingerprint]
		current.lastSent = event.lastSent
		current.suppressed += suppressed
		n.panicEvents[fingerprint] = current
		n.mu.Unlock()
		return err
	}
	return nil
}

func panicFingerprint(component, panicText string, stack []byte) string {
	stackLines := strings.Split(string(stack), "\n")
	if len(stackLines) > 0 && strings.HasPrefix(stackLines[0], "goroutine ") {
		stackLines = stackLines[1:]
	}
	if len(stackLines) > 12 {
		stackLines = stackLines[:12]
	}
	hash := sha256.Sum256([]byte(component + "\x00" + panicText + "\x00" + strings.Join(stackLines, "\n")))
	return hex.EncodeToString(hash[:6])
}

func truncateBytes(data []byte, limit int) []byte {
	if limit <= 0 || len(data) <= limit {
		return data
	}
	return data[:limit]
}

func truncateUTF8(value string, maxBytes int) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	for maxBytes > 0 && (value[maxBytes]&0xc0) == 0x80 {
		maxBytes--
	}
	if maxBytes == 0 {
		return ""
	}
	return value[:maxBytes] + "…"
}

var ErrDisabled = errors.New("通知未启用")
