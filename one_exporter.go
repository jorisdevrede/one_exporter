// Prometheus exporter for OpenNebula.
package main

import (
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/OpenNebula/one/src/oca/go/src/goca"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/spf13/viper"

	"gopkg.in/alecthomas/kingpin.v2"
)

type config struct {
	user     string
	password string
	endpoint string
	interval int
	host     string
	port     int
	path     string
}

var (
	clusterMetrics = make(map[string]*prometheus.GaugeVec)
	hostMetrics    = make(map[string]*prometheus.GaugeVec)
)

func initCollectors() {
	clusterMetrics["TotalMem"] = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "one_cluster_totalmem",
			Help: "total memory available in cluster",
		},[]string{"cluster"})

	clusterMetrics["UsedMem"] = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "one_cluster_usedmem",
			Help: "real used memory in cluster",
		},[]string{"cluster"})

	clusterMetrics["TotalCPU"] = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "one_cluster_totalcpu",
			Help: "total cpu available in cluster",
		},[]string{"cluster"})

	clusterMetrics["UsedCPU"] = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "one_cluster_usedcpu",
			Help: "real used cpu in cluster",
		},[]string{"cluster"})

	clusterMetrics["RunningVMs"] = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "one_cluster_runningvms",
			Help: "running virtual machines in cluster",
		},[]string{"cluster"})

	clusterMetrics["ActiveHosts"] = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "one_cluster_activehosts",
			Help: "succesfully monitored hosts in cluster",
		},[]string{"cluster"})

	hostMetrics["TotalMem"] = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "one_host_totalmem",
			Help: "total memory available on host",
		},[]string{"cluster", "host"})

	hostMetrics["UsedMem"] = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "one_host_usedmem",
			Help: "real used memory on host",
		},[]string{"cluster", "host"})

	hostMetrics["TotalCPU"] = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "one_host_totalcpu",
			Help: "total cpu available on host",
		},[]string{"cluster", "host"})

	hostMetrics["UsedCPU"] = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "one_host_usedcpu",
			Help: "real used cpu on host",
		},[]string{"cluster", "host"})

	hostMetrics["RunningVMs"] = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "one_host_runningvms",
			Help: "running virtual machines on host",
		},[]string{"cluster", "host"})

}

// recordMetrics from OpenNebula
func recordMetrics(config config, logger log.Logger) {

	level.Info(logger).Log("msg", "recording metrics from opennebula frontend", "interval", config.interval)

	client := goca.NewDefaultClient(goca.NewConfig(config.user, config.password, config.endpoint))
	controller := goca.NewController(client)

	for {

		pool, err := controller.Hosts().Info()
		if err != nil {
			level.Error(logger).Log("msg", "error retrieving hosts info", "error", err)
			panic(err)
		}

		type metrics struct {
			cluster, metric string
		}
		sum := make(map[metrics]int)

		for _, host := range pool.Hosts {

			level.Debug(logger).Log("msg", "host metrics",
				"host", host.Name,
				"TotalMem", host.Share.TotalMem,
				"UsedMem", host.Share.UsedMem,
				"TotalCPU", host.Share.TotalCPU,
				"UsedCPU", host.Share.UsedCPU,
				"RunningVMs", host.Share.RunningVMs)

			// record host metrics
			hostMetrics["TotalMem"].With(prometheus.Labels{"cluster": host.Cluster, "host": host.Name}).Set(float64(host.Share.TotalMem))
			hostMetrics["UsedMem"].With(prometheus.Labels{"cluster": host.Cluster, "host": host.Name}).Set(float64(host.Share.UsedMem))
			hostMetrics["TotalCPU"].With(prometheus.Labels{"cluster": host.Cluster, "host": host.Name}).Set(float64(host.Share.TotalCPU))
			hostMetrics["UsedMem"].With(prometheus.Labels{"cluster": host.Cluster, "host": host.Name}).Set(float64(host.Share.UsedMem))
			hostMetrics["RunningVMs"].With(prometheus.Labels{"cluster": host.Cluster, "host": host.Name}).Set(float64(host.Share.RunningVMs))

			// sum cluster metrics
			sum[metrics{host.Cluster, "TotalMem"}] = sum[metrics{host.Cluster, "TotalMem"}] + host.Share.TotalMem
			sum[metrics{host.Cluster, "UsedMem"}] = sum[metrics{host.Cluster, "UsedMem"}] + host.Share.UsedMem
			sum[metrics{host.Cluster, "TotalCPU"}] = sum[metrics{host.Cluster, "TotalCPU"}] + host.Share.TotalCPU
			sum[metrics{host.Cluster, "UsedCPU"}] = sum[metrics{host.Cluster, "UsedCPU"}] + host.Share.UsedCPU
			sum[metrics{host.Cluster, "RunningVMs"}] = sum[metrics{host.Cluster, "RunningVMs"}] + host.Share.RunningVMs

			if host.StateRaw == 2 {
				sum[metrics{host.Cluster, "ActiveHosts"}] = sum[metrics{host.Cluster, "ActiveHosts"}] + 1
			}
		}

		for key, value := range sum {
			// record cluster metrics
			clusterMetrics[key.metric].With(prometheus.Labels{"cluster": key.cluster}).Set(float64(value))
		}

		time.Sleep(time.Duration(config.interval) * time.Second)
	}
}

func newConfig(fileName string, logger log.Logger) config {

	viper.SetDefault("endpoint", "") // "" will be set to "http://localhost:2633/RPC2" by goca
	viper.SetDefault("interval", 60)
	viper.SetDefault("path", "/metrics")
	viper.SetDefault("port", 9621)


	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if fileName != "" {
		level.Info(logger).Log("msg", "using provided configuration file", "file", fileName)

		dir, file := path.Split(fileName)
		viper.SetConfigName(file)
		viper.AddConfigPath(dir)
	}

	err := viper.ReadInConfig()
	if err != nil {
		level.Error(logger).Log("msg", "error reading config file", "error", err)
		panic(err)
	}

	return config{
		user:     viper.Get("user").(string),
		password: viper.Get("password").(string),
		endpoint: "",
		interval: viper.Get("interval").(int),
		host:     viper.Get("host").(string),
		port:     viper.Get("port").(int),
		path:     viper.Get("path").(string),
	}

}

func allowedLevel(logLevel string) level.Option {

	switch strings.ToLower(logLevel) {
	case "error":
		return level.AllowError()
	case "debug":
		return level.AllowDebug()
	default:
		return level.AllowInfo()
	}
}

func main() {

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)

	cfgFile := kingpin.Flag("config", "config file for one_exporter").Short('c').String()
	logLevel := kingpin.Flag("loglevel", "the log level to output. options are error, info or debug. defaults to info").Short('l').Default("info").String()
	kingpin.Parse()

	logger = level.NewFilter(logger, allowedLevel(*logLevel))
	level.Info(logger).Log("msg", "starting exporter for OpenNebula")

	config := newConfig(*cfgFile, logger)
	level.Debug(logger).Log("msg", "loaded config", "user", config.user, "endpoint", config.endpoint)

	initCollectors()

	go recordMetrics(config, logger)

	level.Info(logger).Log("msg", "starting exporter", "host", config.host, "port", config.port, "path", config.path)
	http.Handle(config.path, promhttp.Handler())
	http.ListenAndServe(config.host+":"+strconv.Itoa(config.port), nil)

}
