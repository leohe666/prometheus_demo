package main

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
)

type Callback struct {
	summaryVec *prometheus.SummaryVec
}

func NewCallback(namespace string, subsystem string, name string, help string) *Callback {
	summaryVec := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  namespace,
			Subsystem:  subsystem,
			Name:       name,
			Help:       help,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"type"},
	)
	prometheus.MustRegister(summaryVec)
	return &Callback{
		summaryVec: summaryVec,
	}
}

func (c *Callback) Before(tx *gorm.DB) {
	now := time.Now()
	tx.Set("startTime", now)
}

func (c *Callback) After(tx *gorm.DB, typ string) {
	val, _ := tx.Get("startTime")
	startTime := val.(time.Time)
	duration := time.Since(startTime)
	c.summaryVec.WithLabelValues(typ).Observe(float64(duration.Milliseconds()))
}
