package main

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	devopsClient "github.com/webdevops/azure-devops-exporter/azure-devops-client"
	prometheusCommon "github.com/webdevops/go-prometheus-common"
)

type MetricsCollectorLatestBuild struct {
	CollectorProcessorProject

	prometheus struct {
		build       *prometheus.GaugeVec
		buildStatus *prometheus.GaugeVec
		timeline    *prometheus.GaugeVec
	}
}

func (m *MetricsCollectorLatestBuild) Setup(collector *CollectorProject) {
	m.CollectorReference = collector

	m.prometheus.build = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_devops_build_latest_info",
			Help: "Azure DevOps build (latest)",
		},
		[]string{
			"projectID",
			"buildDefinitionID",
			"buildID",
			"agentPoolID",
			"requestedBy",
			"buildNumber",
			"buildName",
			"sourceBranch",
			"sourceVersion",
			"status",
			"reason",
			"result",
			"url",
		},
	)
	prometheus.MustRegister(m.prometheus.build)

	m.prometheus.buildStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_devops_build_latest_status",
			Help: "Azure DevOps build status (latest)",
		},
		[]string{
			"projectID",
			"buildID",
			"buildNumber",
			"type",
		},
	)
	prometheus.MustRegister(m.prometheus.buildStatus)

	m.prometheus.timeline = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_devops_build_timeline_latest_info",
			Help: "Azure DevOps build timeline (latest)",
		},
		[]string{
			"projectID",
			"buildDefinitionID",
			"buildID",
			"agentPoolID",
			"buildNumber",
			"buildName",
			"sourceBranch",
			"status",
			"buildUrl",
			"reason",
			"result",
			"recordId",
			"recordParentId",
			"recordType",
			"recordName",
			"recordStartTime",
			"recordFinishTime",
			"recordDuration",
			"recordState",
			"recordResult",
			"recordWorkerName",
			"recordOrder",
			"recordErrorCount",
			"recordWarningCount",
			"recordLogUrl",
		},
	)
	prometheus.MustRegister(m.prometheus.timeline)
}

func (m *MetricsCollectorLatestBuild) Reset() {
	m.prometheus.build.Reset()
	m.prometheus.buildStatus.Reset()
	m.prometheus.timeline.Reset()
}

func (m *MetricsCollectorLatestBuild) collectTimelines(buildTimelineMetric *prometheusCommon.MetricList, logger *log.Entry, project devopsClient.Project, build devopsClient.Build) {
	logger.Debug("Query timeline for build '", build.Definition.Name, "'")

	timeline, err := AzureDevopsClient.GetTimeline(project.Id, build.Id)
	if err != nil {
		logger.Error(err)
		return
	}

	for _, record := range timeline.Records {
		buildTimelineMetric.AddInfo(prometheus.Labels{
			"projectID":          project.Id,
			"buildDefinitionID":  int64ToString(build.Definition.Id),
			"buildID":            int64ToString(build.Id),
			"buildNumber":        build.BuildNumber,
			"buildName":          build.Definition.Name,
			"sourceBranch":       build.SourceBranch,
			"agentPoolID":        int64ToString(build.Queue.Pool.Id),
			"status":             build.Status,
			"buildUrl":           build.Url,
			"reason":             build.Reason,
			"result":             build.Result,
			"recordId":           record.Id,
			"recordParentId":     record.ParentId,
			"recordType":         record.Type,
			"recordName":         record.Name,
			"recordStartTime":    record.StartTimestamp,
			"recordFinishTime":   record.FinishTimestamp,
			"recordDuration":     record.Duration(),
			"recordState":        record.State,
			"recordResult":       record.Result,
			"recordWorkerName":   record.WorkerName,
			"recordOrder":        int64ToString(record.Order),
			"recordErrorCount":   int64ToString(record.ErrorCount),
			"recordWarningCount": int64ToString(record.WarningCount),
			"recordLogUrl":       record.Log.Url,
		})
	}
}

func (m *MetricsCollectorLatestBuild) Collect(ctx context.Context, logger *log.Entry, callback chan<- func(), project devopsClient.Project) {
	list, err := AzureDevopsClient.ListLatestBuilds(project.Id)
	if err != nil {
		logger.Error(err)
		return
	}

	buildMetric := prometheusCommon.NewMetricsList()
	buildStatusMetric := prometheusCommon.NewMetricsList()
	buildTimelineMetric := prometheusCommon.NewMetricsList()

	for _, build := range list.List {

		m.collectTimelines(buildTimelineMetric, logger, project, build)

		buildMetric.AddInfo(prometheus.Labels{
			"projectID":         project.Id,
			"buildDefinitionID": int64ToString(build.Definition.Id),
			"buildID":           int64ToString(build.Id),
			"buildNumber":       build.BuildNumber,
			"buildName":         build.Definition.Name,
			"agentPoolID":       int64ToString(build.Queue.Pool.Id),
			"requestedBy":       build.RequestedBy.DisplayName,
			"sourceBranch":      build.SourceBranch,
			"sourceVersion":     build.SourceVersion,
			"status":            build.Status,
			"reason":            build.Reason,
			"result":            build.Result,
			"url":               build.Links.Web.Href,
		})

		buildStatusMetric.AddTime(prometheus.Labels{
			"projectID":   project.Id,
			"buildID":     int64ToString(build.Id),
			"buildNumber": build.BuildNumber,
			"type":        "started",
		}, build.StartTime)

		buildStatusMetric.AddTime(prometheus.Labels{
			"projectID":   project.Id,
			"buildID":     int64ToString(build.Id),
			"buildNumber": build.BuildNumber,
			"type":        "queued",
		}, build.QueueTime)

		buildStatusMetric.AddTime(prometheus.Labels{
			"projectID":   project.Id,
			"buildID":     int64ToString(build.Id),
			"buildNumber": build.BuildNumber,
			"type":        "finished",
		}, build.FinishTime)

		buildStatusMetric.AddDuration(prometheus.Labels{
			"projectID":   project.Id,
			"buildID":     int64ToString(build.Id),
			"buildNumber": build.BuildNumber,
			"type":        "jobDuration",
		}, build.FinishTime.Sub(build.StartTime))
	}

	callback <- func() {
		buildMetric.GaugeSet(m.prometheus.build)
		buildStatusMetric.GaugeSet(m.prometheus.buildStatus)
		buildTimelineMetric.GaugeSet(m.prometheus.timeline)
	}
}
