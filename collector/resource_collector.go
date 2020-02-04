package collector

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/routeros.v2/proto"
)

var uptimeRegex *regexp.Regexp
var uptimeParts [5]time.Duration

func init() {
	uptimeRegex = regexp.MustCompile(`(?:(\d*)w)?(?:(\d*)d)?(?:(\d*)h)?(?:(\d*)m)?(?:(\d*)s)?`)
	uptimeParts = [5]time.Duration{time.Hour * 168, time.Hour * 24, time.Hour, time.Minute, time.Second}
}

type resourceCollector struct {
	props        []string
	descriptions map[string]*prometheus.Desc
}

func newResourceCollector() routerOSCollector {
	c := &resourceCollector{}
	c.init()
	return c
}

func (c *resourceCollector) init() {
	c.props = []string{"free-memory", "total-memory", "cpu-load", "free-hdd-space", "total-hdd-space", "uptime", "board-name", "version"}

	labelNames := []string{"name", "address", "boardname", "version"}
	c.descriptions = make(map[string]*prometheus.Desc)
	for _, p := range c.props {
		c.descriptions[p] = descriptionForPropertyName("system", p, labelNames)
	}
}

func (c *resourceCollector) describe(ch chan<- *prometheus.Desc) {
	for _, d := range c.descriptions {
		ch <- d
	}
}

func (c *resourceCollector) collect(ctx *collectorContext) error {
	stats, err := c.fetch(ctx)
	if err != nil {
		return err
	}

	for _, re := range stats {
		c.collectForStat(re, ctx)
	}

	return nil
}

func (c *resourceCollector) fetch(ctx *collectorContext) ([]*proto.Sentence, error) {
	reply, err := ctx.client.Run("/system/resource/print", "=.proplist="+strings.Join(c.props, ","))
	if err != nil {
		log.WithFields(log.Fields{
			"device": ctx.device.Name,
			"error":  err,
		}).Error("error fetching system resource metrics")
		return nil, err
	}

	return reply.Re, nil
}

func (c *resourceCollector) collectForStat(re *proto.Sentence, ctx *collectorContext) {
	for _, p := range c.props[:6] {
		c.collectMetricForProperty(p, re, ctx)
	}
}

func (c *resourceCollector) collectMetricForProperty(property string, re *proto.Sentence, ctx *collectorContext) {
	var v float64
	var err error
	//	const boardname = "BOARD"
	//	const version = "3.33.3"

	boardname := re.Map["board-name"]
	version := re.Map["version"]

	if property == "uptime" {
		v, err = parseUptime(re.Map[property])
	} else {
		if re.Map[property] == "" {
			return
		}
		v, err = strconv.ParseFloat(re.Map[property], 64)
	}

	if err != nil {
		log.WithFields(log.Fields{
			"device":   ctx.device.Name,
			"property": property,
			"value":    re.Map[property],
			"error":    err,
		}).Error("error parsing system resource metric value")
		return
	}

	desc := c.descriptions[property]
	ctx.ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, v, ctx.device.Name, ctx.device.Address, boardname, version)
}

func parseUptime(uptime string) (float64, error) {
	var u time.Duration

	reMatch := uptimeRegex.FindAllStringSubmatch(uptime, -1)

	// should get one and only one match back on the regex
	if len(reMatch) != 1 {
		return 0, fmt.Errorf("invalid uptime value sent to regex")
	}

	for i, match := range reMatch[0] {
		if match != "" && i != 0 {
			v, err := strconv.Atoi(match)
			if err != nil {
				log.WithFields(log.Fields{
					"uptime": uptime,
					"value":  match,
					"error":  err,
				}).Error("error parsing uptime field value")
				return float64(0), err
			}
			u += time.Duration(v) * uptimeParts[i-1]
		}
	}
	return u.Seconds(), nil
}
