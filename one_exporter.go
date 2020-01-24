// Prometheus exporter for Open Nebula.
package main

import (

	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/OpenNebula/one/src/oca/go/src/goca"
	"github.com/OpenNebula/one/src/oca/go/src/goca/schemas/host"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/spf13/viper"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (

	poolTotalMemGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "one_pool_totalmem",
		Help: "total memory of all hosts in open nebula",
	})
	poolUsedMemGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "one_pool_usedmem",
		Help: "used memory in all hosts in open nebula",
	})
	poolTotalCPUGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "one_pool_totalcpu",
		Help: "total cpu of all hosts in open nebula",
	})
	poolUsedCPUGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "one_pool_usedcpu",
		Help: "used cpu in all hosts in open nebula",
	})
	poolActiveHostsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "one_pool_activehosts",
		Help: "number of active hosts in open nebula",
	})
	poolRunningVMsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "one_pool_runningvms",
		Help: "number of running virtual machines in open nebula",
	})
)

func recordMetrics(pool *host.Pool, logger log.Logger) {

	level.Info(logger).Log("msg", "recording metrics from open nebula frontend")

	for {
		var totalMem int = 0
		var usedMem int = 0
		var totalCPU int = 0
		var usedCPU int = 0
		var runningVMs int = 0

		var activeHosts int = 0

		for _, host := range pool.Hosts {

			level.Debug(logger).Log("msg", "host metrics", "host", host.Name)

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

		time.Sleep(30 * time.Second)
	}
}

func main() {

	var (
		user string
		password string
		host string
		port string
	)

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)

	level.Info(logger).Log("msg", "starting exporter for Open Nebula")

	config := kingpin.Flag("config", "config file for one_exporter").Short('c').String()
	kingpin.Parse()

	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if *config != "" {
		level.Info(logger).Log("msg", "using provided configuration file", "config", *config)

		dir, file := path.Split(*config)
		viper.SetConfigName(file)
		viper.AddConfigPath(dir)
	}

	err := viper.ReadInConfig()
	if err != nil {
		level.Error(logger).Log("msg", "error reading config file", "error", err)
		return
	}

	user = viper.Get("user").(string)
	password = viper.Get("password").(string)
	host = viper.Get("host").(string)
	port = strconv.Itoa(viper.Get("port").(int))

	level.Info(logger).Log("msg", "using config:", "user", user, "host", host, "port", port)

	conf := goca.NewConfig( user, password, "")
	client := goca.NewDefaultClient(conf)
	controller := goca.NewController(client)

	pool, err := controller.Hosts().Info()
	if err != nil {
		level.Error(logger).Log("msg", "error retrieving hosts info", "error", err)
		return
	}

	go recordMetrics(pool, logger)

	level.Info(logger).Log("msg", "starting exporter on endpoint /metrics")
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(host + ":" + port, nil)

}
