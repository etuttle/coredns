# nshorizon

## Name

*nshorizon* - Split-horizon DNS based on the namespace of the client

## Description

This plugin answers queries differently depending on the kubernetes namespace of the dns client.

Say a pod in namespace `my-ns` does a query for `service.internal.example.com`.  If `nshorizon` is
enabled for the `internal.example.com` zone, and the normal lookup process is about to return
`NXDOMAIN`, the plugin will step in and attempt to resolve `service` in the `my-ns` namespace.  If
the service lookup succeeds, a CNAME is returned to the client resolving
`service.internal.example.com` to `service.my-ns.svc.cluster.local`.

## Syntax

**Important:** The POD-MODE must be `pods verified` in the `kubernetes` plugin config.
**Important:** The `cache` plugin must be disabled for nshorizon to work correctly (see Issues).

~~~
nshorizon [ZONE...] @BACKEND-PLUGIN
~~~

* **ZONES** zones which *nshorizon* will attempt service lookups for.
* **BACKEND-PLUGIN** CoreDNS plugin (in @name format) to resolve source IP into namespace.

Currently only @kubernetes is supported.  If a plugin implements the `NsHorizonBackend` interface
then it can be used.

## Metrics

Not yet implemented

## Examples

~~~
nshorizon internal.example.com @kubernetes
~~~

## Known Issues

The `cache` plugin appears to cache TTL 0 records.  Even if the TTL is only increased 5 seconds
(from 0 to 5), it will break split horizon lookups.  Until there is a way to disable the cache
plugin on a per-zone basis, it is not recommended to cache and nshorizon together.
