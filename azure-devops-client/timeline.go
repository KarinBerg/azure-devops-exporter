package AzureDevopsClient

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

type Timeline struct {
	Id                     string           `json:"id"`
	LastChangedByTimestamp string           `json:"lastChangedBy,omitempty"`
	LastChangedOnTimestamp string           `json:"lastChangedOn,omitempty"`
	Records                []TimelineRecord `json:"records"`
	BuildId                int64
}

type Log struct {
	Id   int64  `json:"id"`
	Type string `json:"type"`
	Url  string `json:"url"`
}

type TimelineRecord struct {
	Type            string  `json:"type"`
	Name            string  `json:"name"`
	Id              string  `json:"id"`
	ParentId        string  `json:"parentId"`
	StartTimestamp  string  `json:"startTime,omitempty"`
	FinishTimestamp string  `json:"finishTime,omitempty"`
	State           string  `json:"state"`
	Result          string  `json:"result"`
	WorkerName      string  `json:"workerName"`
	Order           int64   `json:"order"`
	ErrorCount      int64   `json:"errorCount"`
	WarningCount    int64   `json:"warningCount"`
	Log             Log     `json:"log"`
	Issues          []Issue `json:"issues"`
}

type Issue struct {
	Type     string `json:"type"`
	Category string `json:"category"`
	Message  string `json:"message"`
}

func (tr *TimelineRecord) StartTime() *time.Time {
	return parseTime(tr.StartTimestamp)
}

func (tr *TimelineRecord) FinishTime() *time.Time {
	return parseTime(tr.FinishTimestamp)
}

func (tr *TimelineRecord) Duration() string {
	startTime := tr.StartTime()
	finishTime := tr.FinishTime()
	if finishTime != nil && startTime != nil {
		duration := finishTime.Sub(*startTime)
		return duration.String()
	}
	return ""
}

func (c *AzureDevopsClient) GetTimeline(project string, buildId int64) (timeline Timeline, error error) {
	defer c.concurrencyUnlock()
	c.concurrencyLock()

	url := fmt.Sprintf(
		"%v/_apis/build/builds/%v/timeline?api-version=%v",
		url.QueryEscape(project),
		url.QueryEscape(int64ToString(buildId)),
		url.QueryEscape(c.ApiVersion),
	)
	response, err := c.rest().R().Get(url)
	if err := c.checkResponse(response, err); err != nil {
		error = err
		return
	}

	err = json.Unmarshal(response.Body(), &timeline)
	if err != nil {
		error = err
		return
	}
	return
}
