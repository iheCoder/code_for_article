package network

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ============= 3.x Content-Length 不匹配演示 =============
// 说明：
// 1) 声明长度 < 实际发送：handler 只会读取声明长度，多余字节留在连接缓冲。只有当后续字节能组成下一条请求行（含 \r\n）并被解析时才可能触发 400；否则服务器阻塞等待更多数据。
// 2) 多余字节若立刻组成一行但语法非法 => 解析下一请求时立即 400。
// 3) 声明长度 > 实际发送且客户端过早关闭 => handler ReadAll 得到 unexpected EOF。
// 下列测试分别覆盖：
//   - TestHTTPContentLengthExcessDataNoCRLF  (情况1：静默滞留，无 400)
//   - TestHTTPContentLengthExcessDataInvalidLineTriggers400 (情况2：即时 400)
//   - TestHTTPContentLengthShortBodyUnexpectedEOF (情况3：短体 EOF)

// 场景1：多发数据但不足以形成第二请求行 => 不立即 400。
func TestHTTPContentLengthExcessDataNoCRLF(t *testing.T) {
	bodyLenCh := make(chan int, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodyLenCh <- len(b)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	addr := server.Listener.Addr().String()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial 失败: %v", err)
	}
	defer conn.Close()

	declared := 100
	actual := 200
	payload := bytes.Repeat([]byte("A"), actual) // 纯 A 没有 CRLF
	req := fmt.Sprintf("POST /x HTTP/1.1\r\nHost: %s\r\nContent-Length: %d\r\nConnection: keep-alive\r\n\r\n", addr, declared)
	conn.Write([]byte(req))
	conn.Write(payload)

	// 读取首个响应（200 OK），然后很快读不到第二个响应（因为服务器在等“下一行”）
	_ = conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	respBytes, _ := io.ReadAll(conn)

	select {
	case n := <-bodyLenCh:
		if n != declared {
			t.Fatalf("handler 读取=%d 预期=%d", n, declared)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatalf("未收到 handler 读取长度")
	}
	respStr := string(respBytes)
	if !strings.Contains(respStr, "200 OK") {
		t.Fatalf("未看到 200 OK 响应: %q", respStr)
	}
	if strings.Contains(respStr, "400 Bad Request") {
		// 这个场景本应不立即 400
		t.Fatalf("意外出现 400: %q", respStr)
	}
}

// 场景2：多余字节立即构成非法请求行 => 立即 400。
func TestHTTPContentLengthExcessDataInvalidLineTriggers400(t *testing.T) {
	bodyLenCh := make(chan int, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodyLenCh <- len(b)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	addr := server.Listener.Addr().String()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial 失败: %v", err)
	}
	defer conn.Close()

	declared := 5
	body := []byte("ABCDE")
	// 多余部分立刻构成一行（!!!\r\n）=> 解析失败 => 400
	extra := []byte("!!!\r\n")
	req := fmt.Sprintf("POST /y HTTP/1.1\r\nHost: %s\r\nContent-Length: %d\r\nConnection: keep-alive\r\n\r\n", addr, declared)
	conn.Write([]byte(req))
	conn.Write(body)
	conn.Write(extra)

	// 读取直到出现 400 或超时
	deadline := time.Now().Add(800 * time.Millisecond)
	var respData []byte
	buf := make([]byte, 1024)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		n, er := conn.Read(buf)
		if n > 0 {
			respData = append(respData, buf[:n]...)
		}
		if strings.Contains(string(respData), "400 Bad Request") {
			break
		}
		if er != nil {
			break
		}
	}

	select {
	case got := <-bodyLenCh:
		if got != declared {
			t.Fatalf("handler 读取=%d 预期=%d", got, declared)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("未收到 handler 长度")
	}

	raw := string(respData)
	if !strings.Contains(raw, "200 OK") {
		t.Fatalf("未见首个 200 响应 raw=%q", raw)
	}
	if !strings.Contains(raw, "400 Bad Request") {
		t.Fatalf("未观察到 400 (raw=%q)", raw)
	}
}

// 场景3：短体 => unexpected EOF。
func TestHTTPContentLengthShortBodyUnexpectedEOF(t *testing.T) {
	errCh := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, er := io.ReadAll(r.Body)
		errCh <- er
	}))
	defer server.Close()

	addr := server.Listener.Addr().String()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial 失败: %v", err)
	}
	// 不使用 defer Close，提前模拟客户端异常关闭

	declared := 100
	actual := 50
	payload := bytes.Repeat([]byte("B"), actual)
	req := fmt.Sprintf("POST /short HTTP/1.1\r\nHost: %s\r\nContent-Length: %d\r\nConnection: close\r\n\r\n", addr, declared)
	if _, err = conn.Write([]byte(req)); err != nil {
		t.Fatalf("写请求头失败: %v", err)
	}
	if _, err = conn.Write(payload); err != nil {
		t.Fatalf("写请求体失败: %v", err)
	}
	// 直接关闭，服务器读到少于声明长度
	conn.Close()

	select {
	case er := <-errCh:
		if er == nil {
			// 在极少情况下，服务器端可能在连接关闭时映射为通用错误
			// 这里仍视为失败以强调问题。
			t.Fatalf("期望读取错误(短 body) 实际 nil")
		}
		// 通常为 unexpected EOF 或 context canceled 类似。
		if !strings.Contains(er.Error(), "EOF") {
			// 给出提示但不强制失败，可按需要改严格
			if testing.Verbose() {
				fmt.Printf("短体读取返回错误: %v\n", er)
			}
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("未收到 handler 读取结果 (可能 handler 阻塞)")
	}
}
