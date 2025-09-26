package network

import (
	"github.com/gorilla/websocket"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestWebSocketSingleWriterHeartbeat 演示：
// 1. 单独 writer goroutine 负责所有写（广播消息 + 定时 Ping）
// 2. 读 goroutine 负责处理 Pong 并刷新 ReadDeadline（防止幽灵连接）
// 3. 客户端被动接收消息，自动回复 Pong（gorilla/websocket 客户端默认行为）
func TestWebSocketSingleWriterHeartbeat(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	messages := []string{"m1", "m2", "m3", "m4"}
	pongWait := 300 * time.Millisecond
	pingPeriod := 80 * time.Millisecond // 必须 < pongWait

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade err: %v", err)
			return
		}
		outbound := make(chan []byte, 8)
		done := make(chan struct{})

		// reader: 只为处理 Pong & 监控关闭
		conn.SetReadLimit(1024)
		conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(appData string) error {
			conn.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		})
		go func() {
			defer close(done)
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}()

		// writer: 串行化所有写操作 + 定时 Ping
		go func() {
			ticker := time.NewTicker(pingPeriod)
			defer ticker.Stop()
			defer conn.Close()
			for {
				select {
				case msg, ok := <-outbound:
					if !ok {
						return
					}
					conn.SetWriteDeadline(time.Now().Add(100 * time.Millisecond))
					if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
						return
					}
				case <-ticker.C:
					conn.SetWriteDeadline(time.Now().Add(50 * time.Millisecond))
					if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(50*time.Millisecond)); err != nil {
						return
					}
				case <-done:
					return
				}
			}
		}()

		// 模拟业务推送
		go func() {
			for _, m := range messages {
				outbound <- []byte(m)
				time.Sleep(30 * time.Millisecond)
			}
			close(outbound)
		}()
	}))
	defer ts.Close()

	// 客户端连接
	c, _, err := websocket.DefaultDialer.Dial("ws"+ts.URL[len("http"):], nil)
	if err != nil {
		t.Fatalf("dial err: %v", err)
	}
	defer c.Close()

	received := make([]string, 0, len(messages))
	deadlinePerMsg := 200 * time.Millisecond
	for len(received) < len(messages) {
		c.SetReadDeadline(time.Now().Add(deadlinePerMsg))
		_, data, err := c.ReadMessage()
		if err != nil {
			t.Fatalf("read err: %v (received=%v)", err, received)
		}
		received = append(received, string(data))
	}
	for i, m := range messages {
		if received[i] != m {
			t.Fatalf("顺序不符 idx=%d want=%s got=%s", i, m, received[i])
		}
	}
}
