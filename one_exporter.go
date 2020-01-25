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
	poolTotalMemGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "one_pool_totalmem",
		Help: "total memory of all hosts in opennebula",
	})
	poolUsedMemGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "one_pool_usedmem",
		Help: "used memory in all hosts in opennebula",
	})
	poolTotalCPUGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "one_pool_totalcpu",
		Help: "total cpu of all hosts in opennebula",
	})
	poolUsedCPUGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "one_pool_usedcpu",
		Help: "used cpu in all hosts in opennebula",
	})
	poolActiveHostsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "one_pool_activehosts",
		Help: "number of active hosts in opennebula",
	})
	poolRunningVMsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "one_pool_runningvms",
		Help: "number of running virtual machines in opennebula",
	})
)

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

		var totalMem int = 0
		var usedMem int = 0
		var totalCPU int = 0
		var usedCPU int = 0
		var runningVMs int = 0

		var activeHosts int = 0

		for _, host := range pool.Hosts {

			level.Debug(logger).Log("msg", "host metrics",
				"host", host.Name,
				"TotalMem", host.Share.TotalMem,
				"UsedMem", host.Share.UsedMem,
				"TotalCPU", host.Share.TotalCPU,
				"UsedCPU", host.Share.UsedCPU,
				"RunningVMs", host.Share.RunningVMs)

			totalMem = totalMem + host.Share.TotalMem
			usedMem = usedMem + host.Share.UsedMem
			totalCPU = totalCPU + host.Share.TotalCPU
			usedCPU = usedCPU + host.Share.UsedCPU
			runningVMs = runningVMs + host.Share.RunningVMs

			if host.StateRaw == 2 {
				activeHosts = activeHosts + 1
			}
		}

		poolTotalMemGauge.Set(float64(totalMem))
		poolUsedMemGauge.Set(float64(usedMem))
		poolTotalCPUGauge.Set(float64(totalCPU))
		poolUsedCPUGauge.Set(float64(usedCPU))
		poolRunningVMsGauge.Set(float64(runningVMs))

		poolActiveHostsGauge.Set(float64(activeHosts))

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

	go recordMetrics(config, logger)

	level.Info(logger).Log("msg", "starting exporter", "host", config.host, "port", config.port, "path", config.path)
	http.Handle(config.path, promhttp.Handler())
	http.ListenAndServe(config.host+":"+strconv.Itoa(config.port), nil)

}
