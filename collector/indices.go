// Copyright 2021 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus-community/elasticsearch_exporter/pkg/clusterinfo"
	"github.com/prometheus/client_golang/prometheus"
)

type labels struct {
	keys   func(...string) []string
	values func(*clusterinfo.Response, ...string) []string
}

type indexMetric struct {
	Type   prometheus.ValueType
	Desc   *prometheus.Desc
	Value  func(indexStats IndexStatsIndexResponse) float64
	Labels labels
}

type shardMetric struct {
	Type   prometheus.ValueType
	Desc   *prometheus.Desc
	Value  func(data IndexStatsIndexShardsDetailResponse) float64
	Labels labels
}

// Indices information struct
type Indices struct {
	logger            log.Logger
	client            *http.Client
	url               *url.URL
	clusterInfoCh     chan *clusterinfo.Response
	lastClusterInfo   *clusterinfo.Response
	up                prometheus.Gauge
	totalScrapes      prometheus.Counter
	jsonParseFailures prometheus.Counter
	indexMetrics      []*indexMetric
}

// NewIndices defines Indices Prometheus metrics
func NewIndices(logger log.Logger, client *http.Client, url *url.URL) *Indices {

	indexLabels := labels{
		keys: func(...string) []string {
			return []string{"index", "cluster"}
		},
		values: func(lastClusterinfo *clusterinfo.Response, s ...string) []string {
			if lastClusterinfo != nil {
				return append(s, lastClusterinfo.ClusterName)
			}
			// this shouldn't happen, as the clusterinfo Retriever has a blocking
			// Run method. It blocks until the first clusterinfo call has succeeded
			return append(s, "unknown_cluster")
		},
	}

	indices := &Indices{
		logger:        logger,
		client:        client,
		url:           url,
		clusterInfoCh: make(chan *clusterinfo.Response),
		lastClusterInfo: &clusterinfo.Response{
			ClusterName: "unknown_cluster",
		},

		up: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: prometheus.BuildFQName(namespace, "index_stats", "up"),
			Help: "Was the last scrape of the Elasticsearch index endpoint successful.",
		}),
		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Name: prometheus.BuildFQName(namespace, "index_stats", "total_scrapes"),
			Help: "Current total Elasticsearch index scrapes.",
		}),
		jsonParseFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: prometheus.BuildFQName(namespace, "index_stats", "json_parse_failures"),
			Help: "Number of errors while parsing JSON.",
		}),

		indexMetrics: []*indexMetric{
			{
				Type: prometheus.GaugeValue,
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, "indices", "primary_shares_size_gbytes"),
					"Current total size of stored index data in gbytes with only primary shards on all nodes",
					indexLabels.keys(), nil,
				),
				Value: func(indexStats IndexStatsIndexResponse) float64 {
					return float64(indexStats.Primaries.Store.SizeInBytes)
				},
				Labels: indexLabels,
			},
		},
	}

	// start go routine to fetch clusterinfo updates and save them to lastClusterinfo
	go func() {
		_ = level.Debug(logger).Log("msg", "starting cluster info receive loop")
		for ci := range indices.clusterInfoCh {
			if ci != nil {
				_ = level.Debug(logger).Log("msg", "received cluster info update", "cluster", ci.ClusterName)
				indices.lastClusterInfo = ci
			}
		}
		_ = level.Debug(logger).Log("msg", "exiting cluster info receive loop")
	}()
	return indices
}

// ClusterLabelUpdates returns a pointer to a channel to receive cluster info updates. It implements the
// (not exported) clusterinfo.consumer interface
func (i *Indices) ClusterLabelUpdates() *chan *clusterinfo.Response {
	return &i.clusterInfoCh
}

// String implements the stringer interface. It is part of the clusterinfo.consumer interface
func (i *Indices) String() string {
	return namespace + "indices"
}

// Describe add Indices metrics descriptions
func (i *Indices) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range i.indexMetrics {
		ch <- metric.Desc
	}
	ch <- i.up.Desc()
	ch <- i.totalScrapes.Desc()
	ch <- i.jsonParseFailures.Desc()
}

func (i *Indices) fetchAndDecodeIndexStats() (indexStatsResponse, error) {
	var isr indexStatsResponse

	u := *i.url
	u.Path = path.Join(u.Path, "/_all/_stats")

	u.RawQuery = "ignore_unavailable=true&level=shards"

	bts, err := i.queryURL(&u)
	if err != nil {
		return isr, err
	}

	if err := json.Unmarshal(bts, &isr); err != nil {
		i.jsonParseFailures.Inc()
		return isr, err
	}

	return isr, nil
}

func (i *Indices) queryURL(u *url.URL) ([]byte, error) {
	res, err := i.client.Get(u.String())
	if err != nil {
		return []byte{}, fmt.Errorf("failed to get resource from %s://%s:%s%s: %s",
			u.Scheme, u.Hostname(), u.Port(), u.Path, err)
	}

	defer func() {
		err = res.Body.Close()
		if err != nil {
			_ = level.Warn(i.logger).Log(
				"msg", "failed to close http.Client",
				"err", err,
			)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return []byte{}, fmt.Errorf("HTTP Request failed with code %d", res.StatusCode)
	}

	bts, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return []byte{}, err
	}

	return bts, nil
}

// Collect gets Indices metric values
func (i *Indices) Collect(ch chan<- prometheus.Metric) {
	i.totalScrapes.Inc()
	defer func() {
		ch <- i.up
		ch <- i.totalScrapes
		ch <- i.jsonParseFailures
	}()

	// indices
	indexStatsResp, err := i.fetchAndDecodeIndexStats()

	if err != nil {
		i.up.Set(0)
		_ = level.Warn(i.logger).Log(
			"msg", "failed to fetch and decode index stats",
			"err", err,
		)
		return
	}
	i.up.Set(1)

	// Index stats
	for indexName, indexStats := range indexStatsResp.Indices {

		for _, metric := range i.indexMetrics {
			Value, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", metric.Value(indexStats)/1024/1024/1024), 64)
			ch <- prometheus.MustNewConstMetric(
				metric.Desc,
				metric.Type,
				Value,
				metric.Labels.values(i.lastClusterInfo, indexName)...,
			)

		}

	}
}
