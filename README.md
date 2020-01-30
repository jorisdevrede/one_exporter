# ONE exporter

Prometheus exporter for OpenNebula clusters, written in Go. The exporter exposes the metrics of the hosts in a cluster, through the GOCA API of the OpenNebula Frontend. The metrics are provided both per host and summed per cluster to give you a quick overview of your OpenNebula cluster as a whole.


## Use

You can run the one exporter with the `-c` flag pointing to its configuration file. If you don't include a configuration file, the exporter will use the config.yml in the current directory, so `./one_exporter` and `./one_exporter -c config.yml` are identical.

Use the `--help` flag for more information.


## Configuration

The one exporter uses the following parameters in the configuration file.

```yaml

# credentials to access OpenNebula
user: oneadmin
password: oneadmin

# OpenNebula frontend endpoint
# an empty endpoint will default to http://localhost:2633/RPC2
# endpoint:

# frequency to retrieve metrics in seconds. defaults to 60.
# interval: 60

# host and port to run the exporter on
host: 0.0.0.0
port: 9621

# exporter uri to publish on. defaults to /metrics
# path: /metrics

```
