package main

import (
	"hash/fnv"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func hash(s string) float64 {
	h := fnv.New64a()
	return float64(h.Sum64())
}

// NewFullReportScraper creates new scraper for puppet agent report file
func NewFullReportScraper(namespace, reportFilename, disableLockFilename string) PuppetYamlReportScraper {
	return &fullReportScraper{
		reportScraper: newPuppetReportScraper(namespace, reportFilename, disableLockFilename),
	}
}

func (r *fullReportScraper) CollectMetrics(ch chan<- prometheus.Metric) error {
	return r.collectMetrics(ch, r)
}

type fullReportScraper struct {
	*reportScraper
}

func (r *fullReportScraper) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var report fullReport
	if err := unmarshal(&report); err != nil {
		return err
	}

	var configTimestamp float64
	if t, err := time.Parse(time.RFC3339, report.Time); err == nil {
		// Puppet 6 report uses RFC3339 format
		configTimestamp = float64(t.Unix())
	} else if t, err = time.Parse("2006-01-02 15:04:05 -07:00", report.Time); err == nil {
		// Puppet 3 report uses something else
		configTimestamp = float64(t.Unix())
	} else {
		// Fallback to what was used previously
		configTimestamp = hash(report.ConfigurationVersion)
	}

	r.setPuppetVersion(report.PuppetVersion)
	r.setConfigTimestamp(configTimestamp)
	r.setCatalogVersion(report.ConfigurationVersion)

	r.setInfo("environment", report.Environment)

	for sectionName, reportSection := range report.Metrics {
		for _, gauge := range reportSection.Values {
			r.setMetricValue(sectionName, gauge.name, gauge.value)
		}
	}

	return nil
}

type fullReport struct {
	Metrics              fullReportMetricsSections
	PuppetVersion        string `yaml:"puppet_version"`
	ConfigurationVersion string `yaml:"configuration_version"`
	Environment          string
	Time                 string
}

type fullReportMetricsSections map[string]fullReportMetricsSection

type fullReportMetricsSection struct {
	Name   string
	Label  string
	Values []fullReportMetricsSectionValue
}

type fullReportMetricsSectionValue struct {
	name  string
	title string
	value float64
}

func (v *fullReportMetricsSectionValue) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var tuple []string
	if err := unmarshal(&tuple); err != nil {
		return err
	}

	val, err := strconv.ParseFloat(tuple[2], 64)
	if err != nil {
		return err
	}

	v.value = val
	v.name = tuple[0]
	v.title = tuple[1]

	return nil
}
