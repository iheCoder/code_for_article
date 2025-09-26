package network

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// 本文件演示文档中所述“零拷贝”（Go 通过 *os.File -> *net.TCPConn 自动利用 sendfile）的惯用用法，
// 并用基准对比：
// 1. io.Copy (可能走 sendfile)
// 2. 手动小缓冲循环拷贝（必经用户态）
// 3. 手动较大缓冲循环拷贝（减少系统调用但仍非 sendfile）
// 注意：
// - sendfile 是否可用取决于当前平台/内核实现；在某些平台差异可能较小。
// - 基准关注相对区别，不保证数值绝对稳定。
// - HTTP handler 每次都重新打开文件以获得新的偏移（保持语义一致）。

// 结果示例：
// goos: darwin
// goarch: arm64
// pkg: code_for_article/network
// cpu: Apple M2
// BenchmarkZeroCopy_ioCopy-8                 29863             39563 ns/op        424058.58 MB/s      5779 B/op         70 allocs/op
// BenchmarkZeroCopy_manualSmallBuf-8         30681             39480 ns/op        424950.74 MB/s      5779 B/op         70 allocs/op
// BenchmarkZeroCopy_manualLargeBuf-8         30516             39933 ns/op        420138.28 MB/s      5778 B/op         70 allocs/op

var (
	zcOnce     sync.Once
	zcFilePath string
	zcFileSize int64
)

func prepareZeroCopyTestFile(b *testing.B, sizeMB int) {
	zcOnce.Do(func() {
		dir := b.TempDir()
		p := filepath.Join(dir, "big.bin")
		f, err := os.Create(p)
		if err != nil {
			b.Fatalf("create file: %v", err)
		}
		defer f.Close()
		chunk := make([]byte, 1024*1024)
		for i := 0; i < len(chunk); i++ {
			chunk[i] = byte(i % 251)
		}
		for i := 0; i < sizeMB; i++ {
			if _, err := f.Write(chunk); err != nil {
				b.Fatalf("write: %v", err)
			}
		}
		if err := f.Sync(); err != nil {
			b.Fatalf("sync: %v", err)
		}
		info, _ := f.Stat()
		zcFilePath = p
		zcFileSize = info.Size()
	})
}

// 基准公共逻辑：启动服务，复用单客户端连接。
func benchFileServer(b *testing.B, handler http.HandlerFunc) {
	server := httptest.NewServer(handler)
	defer server.Close()
	client := &http.Client{Timeout: 0, Transport: &http.Transport{DisableCompression: true}}
	b.SetBytes(zcFileSize)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			b.Fatalf("get: %v", err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// Handler 使用 io.Copy(w, f) 触发 *os.File -> net.Conn 的 sendfile 路径。
func BenchmarkZeroCopy_ioCopy(b *testing.B) {
	prepareZeroCopyTestFile(b, 16) // 16MB
	h := func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open(zcFilePath)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer f.Close()
		// 关键：直接 io.Copy
		io.Copy(w, f)
	}
	benchFileServer(b, h)
}

// 手动小缓冲循环：增加用户态 <-> 内核态数据复制次数。
func BenchmarkZeroCopy_manualSmallBuf(b *testing.B) {
	prepareZeroCopyTestFile(b, 16)
	bufSize := 4 * 1024 // 4KB
	h := func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open(zcFilePath)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer f.Close()
		buf := make([]byte, bufSize)
		for {
			n, er := f.Read(buf)
			if n > 0 {
				if _, ew := w.Write(buf[:n]); ew != nil {
					return
				}
			}
			if er == io.EOF {
				break
			}
			if er != nil {
				return
			}
		}
	}
	benchFileServer(b, h)
}

// 手动较大缓冲循环：缓解系统调用次数，但仍需一次用户态拷贝。
func BenchmarkZeroCopy_manualLargeBuf(b *testing.B) {
	prepareZeroCopyTestFile(b, 16)
	bufSize := 256 * 1024 // 256KB
	h := func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open(zcFilePath)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer f.Close()
		buf := make([]byte, bufSize)
		for {
			n, er := f.Read(buf)
			if n > 0 {
				if _, ew := w.Write(buf[:n]); ew != nil {
					return
				}
			}
			if er == io.EOF {
				break
			}
			if er != nil {
				return
			}
		}
	}
	benchFileServer(b, h)
}

// 可选：模拟禁用 sendfile 的情况（部分平台变量或包装会剥夺优化）。
// 这里只提供结构；若需强制退化，可在 Linux 下设置环境变量：GODEBUG=asyncpreemptoff=1 (非直接相关) 或修改源码包装。留空作注释说明。
// func BenchmarkZeroCopy_wrappedReader(b *testing.B) { ... }

// 附：演示非基准的简单用法（供文档引用）。
func Example_zeroCopy() {
	// 假设在真实服务器 handler 中：
	// func handler(w http.ResponseWriter, r *http.Request) {
	//   f, _ := os.Open("/path/to/file")
	//   defer f.Close()
	//   // 简洁写法：可能触发零拷贝 sendfile
	//   io.Copy(w, f)
	// }
	// Output:
}

// 额外：避免基准运行时间过短导致噪音，可加最小时间
func init() { testing.Init(); _ = time.Now() }
