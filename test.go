package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"sync/atomic" // 导入 atomic 包用于原子操作
	"time"
)

const (
	// targetURL   = "http://localhost:8080/test" // 目标 URL
	// targetURL   = "http://localhost:8080/gorm"  // 目标 URL
	targetURL   = "http://localhost:8080/redis" // 目标 URL
	numRequests = 1000                          // 总请求次数
	concurrency = 60                            // 最大并发协程数量
)

func main() {
	log.Printf("🚀 开始向 %s 并发发送 %d 次 HTTP GET 请求，最大并发数: %d...", targetURL, numRequests, concurrency)

	var (
		wg        sync.WaitGroup // 用于等待所有协程完成
		successes int32          // 原子计数器：成功请求数
		failures  int32          // 原子计数器：失败请求数
	)

	// 创建一个 HTTP 客户端实例，并设置请求超时时间
	// 建议使用单个客户端实例，因为它内部有连接池，可以复用连接
	client := &http.Client{
		Timeout: 5 * time.Second, // 每个请求的超时时间
	}

	// 使用 channel 来限制并发 goroutine 的数量
	// channel 的容量就是允许同时运行的协程数量
	guard := make(chan struct{}, concurrency)

	startTime := time.Now() // 记录开始时间

	for i := 0; i < numRequests; i++ {
		wg.Add(1)           // 每启动一个协程，Waitgroup 计数加 1
		guard <- struct{}{} // 尝试向 channel 发送一个空结构体，如果 channel 已满，这里会阻塞，直到有“槽位”释放
		time.Sleep(time.Millisecond * 100)
		go func(requestNum int) {
			defer wg.Done()            // 协程完成时，Waitgroup 计数减 1
			defer func() { <-guard }() // 协程执行完毕后，从 channel 接收一个值，释放一个“槽位”

			// 发送 HTTP GET 请求
			resp, err := client.Get(targetURL)
			if err != nil {
				atomic.AddInt32(&failures, 1) // 使用原子操作增加失败计数
				// log.Printf("请求 %d 失败: %v", requestNum, err) // 调试时可以取消注释查看详细失败信息
				return
			}
			defer resp.Body.Close() // 确保关闭响应体，防止资源泄漏

			// 检查 HTTP 状态码
			if resp.StatusCode != http.StatusOK {
				atomic.AddInt32(&failures, 1) // 使用原子操作增加失败计数
				// log.Printf("请求 %d 返回非 200 状态码: %d", requestNum, resp.StatusCode)
				return
			}

			// 读取响应体（可选：如果不需要响应内容，可以省略这部分以节省资源）
			_, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				atomic.AddInt32(&failures, 1) // 使用原子操作增加失败计数
				// log.Printf("请求 %d 读取响应体失败: %v", requestNum, err)
				return
			}

			atomic.AddInt32(&successes, 1) // 使用原子操作增加成功计数
			// log.Printf("请求 %d 成功", requestNum) // 调试时可以取消注释查看每个成功请求
		}(i + 1) // 传入请求编号
	}

	wg.Wait() // 等待所有 goroutine 完成其任务

	totalTime := time.Since(startTime) // 计算总耗时

	log.Println("--- 并发请求完成 ---")
	log.Printf("📊 总请求数: %d", numRequests)
	log.Printf("✅ 成功请求: %d", atomic.LoadInt32(&successes)) // 读取原子计数器的值
	log.Printf("❌ 失败请求: %d", atomic.LoadInt32(&failures))  // 读取原子计数器的值
	log.Printf("⏱️ 总耗时: %v", totalTime)
	if numRequests > 0 {
		log.Printf("⚡ 平均每秒请求数 (QPS): %.2f", float64(numRequests)/totalTime.Seconds())
		log.Printf("⏳ 平均每个请求耗时: %v", totalTime/time.Duration(numRequests))
	} else {
		log.Println("没有发送任何请求。")
	}
}
