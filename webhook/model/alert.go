package model

import "time"

type Alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:annotations`
	StartsAt    time.Time         `json:"startsAt"`
	EndsAt      time.Time         `json:"endsAt"`
}

type Notification struct {
	Version           string            `json:"version" form:"version"`
	GroupKey          string            `json:"groupKey" form:"group_key"`
	Status            string            `json:"status" form:"status"`
	Receiver          string            `json:"receiver" form:"receiver"`
	GroupLabels       map[string]string `json:"groupLabels" form:"group_labels"`
	CommonLabels      map[string]string `json:"commonLabels"  form:"common_labels"`
	CommonAnnotations map[string]string `json:"commonAnnotations" form:"common_annotations"`
	ExternalURL       string            `json:"externalURL" form:"external_url"`
	Alerts            []Alert           `json:"alerts" form:"alerts"`
}
