package main

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	devopsClient "github.com/webdevops/azure-devops-exporter/azure-devops-client"
	prometheusCommon "github.com/webdevops/go-prometheus-common"
)

type MetricsCollectorBuild struct {
	CollectorProcessorProject

	prometheus struct {
		build       *prometheus.GaugeVec
		buildStatus *prometheus.GaugeVec
		timeline    *prometheus.GaugeVec

		buildDefinition *prometheus.GaugeVec

		buildTimeProject *prometheus.SummaryVec
		jobTimeProject   *prometheus.SummaryVec
	}
}

func (m *MetricsCollectorBuild) Setup(collector *CollectorProject) {
	m.CollectorReference = collector

	m.prometheus.build = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_devops_build_info",
			Help: "Azure DevOps build",
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
			Name: "azure_devops_build_status",
			Help: "Azure DevOps build status",
		},
		[]string{
			"projectID",
			"buildID",
			"buildDefinitionID",
			"buildNumber",
			"type",
			"result",
			"sourceBranch",
		},
	)
	prometheus.MustRegister(m.prometheus.buildStatus)

	m.prometheus.buildDefinition = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_devops_build_definition_info",
			Help: "Azure DevOps build definition",
		},
		[]string{
			"projectID",
			"buildDefinitionID",
			"buildNameFormat",
			"buildDefinitionName",
			"path",
			"url",
		},
	)
	prometheus.MustRegister(m.prometheus.buildDefinition)

	m.prometheus.timeline = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_devops_build_timeline_info",
			Help: "Azure DevOps build timeline",
		},
		[]string{
			"projectID",
			"buildDefinitionID",
			"buildID",
			"agentPoolID",
			"buildNumber",
			"buildName",
			"status",
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

func (m *MetricsCollectorBuild) Reset() {
	m.prometheus.build.Reset()
	m.prometheus.buildDefinition.Reset()
	m.prometheus.buildStatus.Reset()
	m.prometheus.timeline.Reset()
}

func (m *MetricsCollectorBuild) Collect(ctx context.Context, logger *log.Entry, callback chan<- func(), project devopsClient.Project) {
	m.collectDefinition(ctx, logger, callback, project)
	m.collectBuilds(ctx, logger, callback, project)
}

func (m *MetricsCollectorBuild) collectDefinition(ctx context.Context, logger *log.Entry, callback chan<- func(), project devopsClient.Project) {
	list, err := AzureDevopsClient.ListBuildDefinitions(project.Id)
	if err != nil {
		logger.Error(err)
		return
	}

	buildDefinitonMetric := prometheusCommon.NewMetricsList()

	for _, buildDefinition := range list.List {
		buildDefinitonMetric.Add(prometheus.Labels{
			"projectID":           project.Id,
			"buildDefinitionID":   int64ToString(buildDefinition.Id),
			"buildNameFormat":     buildDefinition.BuildNameFormat,
			"buildDefinitionName": buildDefinition.Name,
			"path":                buildDefinition.Path,
			"url":                 buildDefinition.Links.Web.Href,
		}, 1)
	}

	callback <- func() {
		buildDefinitonMetric.GaugeSet(m.prometheus.buildDefinition)
	}
}

func (m *MetricsCollectorBuild) collectTimelines(buildTimelineMetric *prometheusCommon.MetricList, logger *log.Entry, project devopsClient.Project, build devopsClient.Build) {
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
			"agentPoolID":        int64ToString(build.Queue.Pool.Id),
			"status":             build.Status,
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
			"recordLogUrl":       record.LogUrl,
		})
	}
}

func (m *MetricsCollectorBuild) collectBuilds(ctx context.Context, logger *log.Entry, callback chan<- func(), project devopsClient.Project) {
	minTime := time.Now().Add(-opts.Limit.BuildHistoryDuration)

	list, err := AzureDevopsClient.ListBuildHistory(project.Id, minTime)
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

		buildStatusMetric.AddBool(prometheus.Labels{
			"projectID":         project.Id,
			"buildID":           int64ToString(build.Id),
			"buildDefinitionID": int64ToString(build.Definition.Id),
			"buildNumber":       build.BuildNumber,
			"type":              "succeeded",
			"result":            build.Result,
			"sourceBranch":      build.SourceBranch,
		}, build.Result == "succeeded")

		buildStatusMetric.AddTime(prometheus.Labels{
			"projectID":         project.Id,
			"buildID":           int64ToString(build.Id),
			"buildDefinitionID": int64ToString(build.Definition.Id),
			"buildNumber":       build.BuildNumber,
			"type":              "queued",
			"result":            build.Result,
			"sourceBranch":      build.SourceBranch,
		}, build.QueueTime)

		buildStatusMetric.AddTime(prometheus.Labels{
			"projectID":         project.Id,
			"buildID":           int64ToString(build.Id),
			"buildDefinitionID": int64ToString(build.Definition.Id),
			"buildNumber":       build.BuildNumber,
			"type":              "started",
			"result":            build.Result,
			"sourceBranch":      build.SourceBranch,
		}, build.StartTime)

		buildStatusMetric.AddTime(prometheus.Labels{
			"projectID":         project.Id,
			"buildID":           int64ToString(build.Id),
			"buildDefinitionID": int64ToString(build.Definition.Id),
			"buildNumber":       build.BuildNumber,
			"type":              "finished",
			"result":            build.Result,
			"sourceBranch":      build.SourceBranch,
		}, build.FinishTime)

		buildStatusMetric.AddDuration(prometheus.Labels{
			"projectID":         project.Id,
			"buildID":           int64ToString(build.Id),
			"buildDefinitionID": int64ToString(build.Definition.Id),
			"buildNumber":       build.BuildNumber,
			"type":              "jobDuration",
			"result":            build.Result,
			"sourceBranch":      build.SourceBranch,
		}, build.FinishTime.Sub(build.StartTime))
	}

	callback <- func() {
		buildMetric.GaugeSet(m.prometheus.build)
		buildStatusMetric.GaugeSet(m.prometheus.buildStatus)
		buildTimelineMetric.GaugeSet(m.prometheus.timeline)
	}
}
