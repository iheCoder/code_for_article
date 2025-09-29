package kernal

// 该测试演示通过文件描述符继承实现监听套接字劫持（零停机交接）的最小可运行示例。
// 核心步骤：
// 1. 父进程创建 TCP 监听器并处理一次请求（返回父 PID）。
// 2. 父进程取得监听器底层 FD (dup 后的 *os.File) ，通过 StartProcess 的 ExtraFiles 传给子进程。
//    按约定子进程中该 FD 号为 3。
// 3. 启动子进程（设置环境变量 CHILD=1），子进程用 fd=3 复原 listener，Accept 一次并返回子 PID。
// 4. 父进程关闭自身 listener（不影响子进程继承的 FD），再次 Dial，应拿到子 PID，且与父 PID 不同。
// 5. 证明同一个内核监听套接字被新进程无缝接管，期间端口未释放、无竞争重绑定。
//
// 注意：
// - 仅在类 Unix 系统验证；在 Windows 上直接跳过。
// - 为简化演示，父子各自只处理一次请求；真实生产应使用 HTTP 服务器并优雅耗尽旧连接。
// - net.TCPListener.File() 返回的是 dup 的 FD；原 listener 与 dup 引用同一 open file description。
//   传给子进程的是 dup；关闭父 listener 不影响子进程持有的 dup FD。

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestFDInheritanceSocketHijack(t *testing.T) {
	// 子进程执行路径：通过 fd=3 接管监听并服务一次后退出。
	if os.Getenv("CHILD") == "1" {
		childServeOnce()
		// childServeOnce 内部会 os.Exit，这里理论上不会到达。
		return
	}

	if runtime.GOOS == "windows" {
		t.Skip("FD 继承示例不适用于 Windows 平台")
	}

	// 1. 父进程创建监听器
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("父进程监听失败: %v", err)
	}
	defer ln.Close()
	addr := ln.Addr().String()

	parentPID := os.Getpid()

	// 接收一次连接并返回 parent:PID\n
	parentAcceptedCh := make(chan struct{})
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return // 关闭或错误直接结束
		}
		fmt.Fprintf(conn, "parent:%d\n", parentPID)
		conn.Close()
		close(parentAcceptedCh)
		// 继续阻塞等待，直到 listener 被关闭（演示旧进程“停止接收”）
		for {
			if _, err := ln.Accept(); err != nil {
				return
			}
		}
	}()

	// 初始请求应由父进程响应
	got := mustDialReadLine(t, addr)
	if got != fmt.Sprintf("parent:%d", parentPID) {
		t.Fatalf("期望父 PID 响应 %d, 得到 %q", parentPID, got)
	}
	<-parentAcceptedCh // 确保父进程完成第一次 Accept

	// 2. 取得底层 FD (dup)
	tcpLn, ok := ln.(*net.TCPListener)
	if !ok {
		t.Fatalf("listener 不是 *net.TCPListener")
	}
	f, err := tcpLn.File() // dup 一个 FD
	if err != nil {
		t.Fatalf("获取底层文件描述符失败: %v", err)
	}
	// 不要提前关闭 f，需传给子进程；父进程稍后关闭原 listener

	// 3. 启动子进程并传递 FD (位于 ExtraFiles[0] => 子进程 fd=3)
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("获取可执行文件失败: %v", err)
	}

	env := append(os.Environ(), "CHILD=1", "HIJACK_ADDR="+addr)

	// 使用 go test 生成的测试二进制；只执行本测试一次即可，子进程通过 CHILD 环境变量分支逻辑。
	cmd := exec.Cmd{
		Path:       exe,
		Args:       []string{exe, "-test.run=TestFDInheritanceSocketHijack"},
		Env:        env,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		ExtraFiles: []*os.File{f}, // fd=3
	}

	if err = cmd.Start(); err != nil {
		t.Fatalf("启动子进程失败: %v", err)
	}

	// 等待子进程进入 Accept 状态（简单 sleep；更严格可用管道同步）
	time.Sleep(150 * time.Millisecond)

	// 4. 关闭父进程 listener -> 不再接受新连接（旧连接仍可处理，这里简化为已完成）
	if err := ln.Close(); err != nil {
		t.Fatalf("关闭父监听器失败: %v", err)
	}

	// 5. 再次请求，应命中子进程并返回 child:PID
	var childLine string
	retryDeadline := time.Now().Add(2 * time.Second)
	for {
		childLine, err = tryDialReadLine(addr)
		if err == nil && strings.HasPrefix(childLine, "child:") {
			break
		}
		if time.Now().After(retryDeadline) {
			t.Fatalf("在超时时间内未获得子进程响应，最后错误: %v, 最后读取: %q", err, childLine)
		}
		time.Sleep(50 * time.Millisecond)
	}

	parts := strings.Split(strings.TrimSpace(childLine), ":")
	if len(parts) != 2 {
		t.Fatalf("子进程响应格式错误: %q", childLine)
	}
	childPID, _ := strconv.Atoi(parts[1])
	if childPID == parentPID {
		t.Fatalf("期望来自不同进程的 PID, 但得到同一 PID %d", childPID)
	}

	// 等待子进程退出
	if err := cmd.Wait(); err != nil {
		t.Fatalf("子进程退出错误: %v", err)
	}
}

// childServeOnce 作为子进程入口：用 fd=3 复原监听器并服务一次。
func childServeOnce() {
	addr := os.Getenv("HIJACK_ADDR")
	f := os.NewFile(uintptr(3), "inherited-listener")
	ln, err := net.FileListener(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[child]%d 复原监听器失败: %v\n", os.Getpid(), err)
		os.Exit(2)
	}
	// Accept 1 个连接并返回 child:PID
	conn, err := ln.Accept()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[child]%d Accept 失败: %v (addr=%s)\n", os.Getpid(), err, addr)
		os.Exit(3)
	}
	fmt.Fprintf(conn, "child:%d\n", os.Getpid())
	conn.Close()
	ln.Close()
	os.Exit(0)
}

func mustDialReadLine(t *testing.T, addr string) string {
	line, err := tryDialReadLine(addr)
	if err != nil {
		t.Fatalf("请求 %s 失败: %v", addr, err)
	}
	return line
}

func tryDialReadLine(addr string) (string, error) {
	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	if err := conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		return "", err
	}
	r := bufio.NewReader(conn)
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return "", errors.New("空响应")
	}
	return line, nil
}
