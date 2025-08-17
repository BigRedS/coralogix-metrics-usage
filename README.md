# coralogix-metrics-usage

Another script for checking out which metrics are used in Coralogix, and by how much.

It will grab your dashboards and alerts, and produce a bunch of YAML listing which metric names and labels are used, and by which boards and alerts; see the included 'output.yaml' file for an example.

The top level of the YAML will be a metric name, each metric name has a series of labels, and each label contains a list of dashboards and of metrics that use it; when a metric is used without specifying a label, it is recorded as if the label was '_':

```
metric1:
  somelabel:
    dashboards:
      <dashboard id>: <query>
      <dashboard id>: <query>
    alerts:
      <alert id>: <query>
  someotherlabel:
    alerts:
      <alert id>: <query>
metric2:
  _:
    dashboards:
      <dashboard_id>: query
````

## Limitations

This is very much unfinished:

* 'parsing' is regex-based, not a proper parser. Funny queries will be missed :/ The `promql-parser` library tries to validate the correctness of the query first, and often falsely judges them as invalid.

* Some forms of query are not well-processed. Especially, currently, those of the form `sum by (k8s_cluster_name,cloud_account_id,k8s_daemonset_name)({__name__=~"k8s_daemonset_misscheduled_nodes__node_",k8s_namespace_name=~".+"})` where the metric names are the result of that name match.

* Alerts and Dashboards aren't the only places you might use metrics.

Each of these might improve with time!

# Installation and running

On something Debian-like you'll need to `apt-get install python3 python3-dotenv`

On anything else, get Python installed and do ```pip install -r requirements.txt```

There are no supported arguments; everything is set via environment veriables:

Required:
  * `CX_API_KEY` - your Coralogix API key for API access. Should be a Personal key, not a send-your-data one.
  * `CX_REGION`  - the three-character name of the region your CX is in (EU0, EU2, etc.)

The expectation is that these are set in a .env file but you can set the env vars manually, too.
