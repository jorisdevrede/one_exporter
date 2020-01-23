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
		Help: "combined total memory of active hosts in open nebula",
	})
	poolFreeMemGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "one_pool_freemem",
		Help: "combined free memory of active hosts in open nebula",
	})
	poolTotalCPUGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "one_pool_totalcpu",
		Help: "combined total cpu of active hosts in open nebula",
	})
	poolFreeCPUGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "one_pool_freecpu",
		Help: "combined free memory of active hosts in open nebula",
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

func recordMetrics(pool *host.Pool) {

	for {
		var totalMem int = 0
		var freeMem int = 0
		var totalCPU int = 0
		var freeCPU int = 0
		var runningVMs int = 0

		var activeHosts int = 0

		for _, host := range pool.Hosts {

			// state 2 is monitored which means active
			if host.StateRaw == 2 {

				totalMem = totalMem + host.Share.TotalMem
				freeMem = freeMem + host.Share.FreeMem
				totalCPU = totalCPU + host.Share.TotalCPU
				freeCPU = freeCPU + host.Share.FreeCPU
				runningVMs = runningVMs + host.Share.RunningVMs

				activeHosts = activeHosts + 1

			}
		}

		poolTotalMemGauge.Set(float64(totalMem))
		poolFreeMemGauge.Set(float64(freeMem))
		poolTotalCPUGauge.Set(float64(totalCPU))
		poolFreeCPUGauge.Set(float64(freeCPU))
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

	go recordMetrics(pool)

	level.Info(logger).Log("msg", "starting exporter on endpoint /metrics")
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(host + ":" + port, nil)

}
