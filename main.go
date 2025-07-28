package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand/v2"
	"net/http"
	"prometheus_demo/webhook/handler"
	"prometheus_demo/webhook/model"
	"prometheus_demo/webhook/util"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/gorm"
)

func initPrometheus() {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		// 这里配置prometheus的配置
		http.ListenAndServe(":8081", nil)
	}()
}

func initDb() *gorm.DB {
	db, err := gorm.Open(mysql.Open("root:123456@tcp(localhost:3306)/mall"))
	if err != nil {
		panic(err)
	}
	var Callback = NewCallback("my_namespace",
		"my_subsystem",
		"gorm_test",
		"统计GORM执行时间")
	err = db.Callback().Query().Before("*").Register("prometheus_query_before", func(tx *gorm.DB) {
		Callback.Before(tx)
	})

	err = db.Callback().Query().After("*").Register("prometheus_query_after", func(tx *gorm.DB) {
		Callback.After(tx, "query")
	})
	return db
}

// User 对应数据库中的user表
type User struct {
	Id   int64  `gorm:"primaryKey,autoIncrement"`
	Name string `gorm:"type=varchar(128)"`
}

func VectorSummary() {
	vec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "my_namespace",
		Subsystem: "my_subsystem",
		Name:      "my_summary_vec",
		ConstLabels: map[string]string{
			"server":  "localhost:8080",
			"env":     "test",
			"appname": "test_app",
		},
	}, []string{"pattern", "method", "status"})
	prometheus.MustRegister(vec)
	vec.WithLabelValues("/user/:id", "POST", "200").Observe(128)
}

func initRedis() redis.Cmdable {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	hook := NewPrometheusHook("my_namespace", "my_subsystem", "redis_test", "统计redis执行时间")
	client.AddHook(hook)
	return client
}

func main() {
	initPrometheus()
	db := initDb()
	client := initRedis()

	server := gin.Default()

	// 使用SummaryBuilder中间件来收集指标
	server.Use(NewSummaryBuilder("my_namespace",
		"my_subsystem",
		"test",
		"统计响应时间").Build())

	// 使用NewGaugeBuilder中间件来收集指标
	server.Use(NewGaugeBuilder("my_namespace",
		"my_subsystem",
		"http_req",
		"统计当前活跃的请求数").Build())

	// query:my_namespace_my_subsystem_test 统计接口响应时间
	// query:my_namespace_my_subsystem_http_req_active_req 统计当前活跃的请求数量
	server.GET("/test", func(ctx *gin.Context) {
		num := rand.IntN(3)
		time.Sleep(time.Duration(num) * time.Second)
		fmt.Println("num", num)
		ctx.String(http.StatusOK, "OK")
	})

	// query:my_namespace_my_subsystem_gorm_test 统计 GORM 执行时间
	server.GET("/gorm", func(ctx *gin.Context) {
		// 执行数据库操作
		var users []User
		err := db.WithContext(ctx).Find(&users).Error
		if err != nil {
			ctx.JSON(http.StatusOK, gin.H{
				"code": 500,
				"msg":  "系统错误",
			})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{
			"code": 200,
			"data": users,
			"msg":  "OK",
		})
	})

	// query:my_namespace_my_subsystem_redis_test 统计 Redis 执行时间
	server.GET("/redis", func(ctx *gin.Context) {
		// 执行redis操作
		err := client.Set(ctx, "test_key", []byte("aaaaaa"), 5*time.Second).Err()
		if err != nil {
			ctx.JSON(http.StatusOK, gin.H{
				"code": 500,
				"msg":  "系统错误",
			})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{
			"code": 200,
			"msg":  "OK",
		})
	})
	server.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	server.POST("/webhook", func(c *gin.Context) {
		// 1. 在任何绑定操作之前，首先读取原始请求体
		bodyBytes, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			log.Printf("Error reading raw body: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
			return
		}

		// 打印原始请求体
		log.Printf("Raw Request Body: %s", string(bodyBytes))
		// 注意这里的结构中的参数是通过prometheus中rules中定义的,见:https://github.com/leohe666/gonivinck_copy/blob/gonivinck_copy_self/prometheus/rules/cpu.rules中的 remark字段
		var req model.Notification
		if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误"})
			return
		}
		larkRequest, _ := util.TransformToLarkRequest(req)
		bytesData, _ := json.Marshal(larkRequest)
		reqr, _ := http.NewRequest("POST", handler.LARK_URL, bytes.NewReader(bytesData))
		reqr.Header.Add("content-type", "application/json")
		res, err := http.DefaultClient.Do(reqr)
		// 飞书服务器可能通信失败
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"error": err.Error(),
			})

			return
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		var larkResponse model.LarkResponse
		err = json.Unmarshal([]byte(body), &larkResponse)
		// 飞书服务器返回的包可能有问题
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"error": err.Error(),
			})

			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "successful receive alert notification message!",
		})
	})

	server.Run(":8080")
}
