package main

// 响应时间
import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

// SummaryBuilder 是一个用于构建Prometheus Summary指标的结构体
type SummaryBuilder struct {
	namespace string
	subsystem string
	name      string
	help      string
}

// NewSummaryBuilder 创建一个新的SummaryBuilder实例
func NewSummaryBuilder(namespace string, subsystem string, name string, help string) *SummaryBuilder {
	return &SummaryBuilder{
		namespace: namespace,
		subsystem: subsystem,
		name:      name,
		help:      help,
	}
}

// Build 构建Gin中间件函数，用于收集请求响应时间的指标
func (builder *SummaryBuilder) Build() gin.HandlerFunc {
	summaryVec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:   builder.namespace,
		Subsystem:   builder.subsystem,
		Name:        builder.name,
		Help:        builder.help,
		ConstLabels: map[string]string{},
		Objectives: map[float64]float64{
			0.5:   0.01,
			0.75:  0.01,
			0.90:  0.005,
			0.99:  0.001,
			0.999: 0.0001,
		},
	}, []string{"pattern", "method", "status"})
	prometheus.MustRegister(summaryVec)
	return func(ctx *gin.Context) {
		now := time.Now()
		defer func() {
			duration := time.Since(now)
			fmt.Println("duration", duration)
			var pattern = ctx.FullPath()
			var method = ctx.Request.Method
			var status = ctx.Writer.Status()
			summaryVec.WithLabelValues(pattern, method, strconv.Itoa(status)).Observe(float64(duration.Microseconds()))
		}()
		ctx.Next()
	}
}
