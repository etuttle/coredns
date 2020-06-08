# nshorizon

## Name

*nshorizon* - Split-horizon DNS based on the namespace of the client

## Description

This plugin answers queries differently depending on the kubernetes namespace of the dns client.

In an example scenario, `nshorizon` is enabled for zone `internal.example.org`.  A pod in namespace `my-ns` does a query for `service.internal.example.org`.  CoreDNS does the usual lookup process for `service.internal.example.org`.  If a record exists, `nshorizon` will not alter the response; however if the usual lookup returns `NXDOMAIN`, `nslookup` will step in and look for a `service` service in the `my-ns` namespace.  If the service lookup succeeds, a CNAME is returned to the client resolving `service.internal.example.com` to `service.my-ns.svc.cluster.local`.

## Syntax

~~~
nshorizon [ZONE...] @BACKEND-PLUGIN
~~~

* **ZONES** zones which *nshorizon* will resolve for.  **Ensure that the `cache` plugin is not enabled** for the same zones (see Known Issues) 
* **BACKEND-PLUGIN** CoreDNS plugin (in @name format) which provide source-ip to namespace lookups.
  Currently @kubernetes is supported, with conditions:
    * POD-MODE must be `pods verified` in the `kubernetes` plugin config.  See [Metadata](../kubernetes/README.md#metadata) in the `kubernetes` plugin README.

## Metrics

Not yet implemented

## Examples

~~~
nshorizon internal.example.com @kubernetes
~~~

## Known Issues

The `cache` plugin appears to cache TTL 0 records.  If a record is cached for a zone that nshorizon is resolving, it will cause records to leak across namespaces.
