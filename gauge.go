package main

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

type GaugeBuilder struct {
	namespace string
	subsystem string
	name      string
	help      string
}

func NewGaugeBuilder(namespace string, subsystem string, name string, help string) *GaugeBuilder {
	return &GaugeBuilder{
		namespace: namespace,
		subsystem: subsystem,
		name:      name,
		help:      help,
	}
}

func (builder *GaugeBuilder) Build() gin.HandlerFunc {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: builder.namespace,
		Subsystem: builder.subsystem,
		Name:      builder.name + "_active_req",
		Help:      builder.help,
	})
	prometheus.MustRegister(gauge)
	return func(ctx *gin.Context) {
		gauge.Inc()
		defer gauge.Dec()
		ctx.Next()
	}
}
