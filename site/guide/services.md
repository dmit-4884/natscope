---
title: Services
description: Find NATS Micro services, see their health, endpoints and instances, and call an endpoint from Request / Reply.
---

# Services

**Services** in the sidebar lists the request/reply services built with NATS Micro, the service
framework in the NATS clients (Go, JavaScript, Python, Rust and others), that answer on the current
connection. Natscope asks them on `$SRV.INFO`, as `nats micro ls` does, and on `$SRV.STATS` for their
statistics, every 5 seconds, and waits up to 2 seconds for the answers. Answers that aren't valid service
replies are left out.

Try it with the demo service that ships with the NATS CLI: `nats micro serve demo`.

## The list

The list on the left shows each service with its instances and versions, and, when statistics are
available, its request rate, error share and health:

- **Healthy**: fewer than 1% of the requests failed.
- **Degraded**: more than 1% failed.
- **Failing**: more than 10% failed.
- **Idle**: no requests.

Rates and health cover the time between the last two refreshes, so they appear after the second one.
The header counts services, instances and failing services. **Auto-refresh** asks again every 5
seconds; **Refresh** asks at once. Filter the list by service name, description, endpoint name or
subject.

Each service has its own address, such as `/services/orders`, so Back and bookmarks work.

## A service

Click a service to see:

- Its request rate, error share and average time for the last window. Hover them for the totals since
  the instances started.
- **Endpoints**: the subject each one listens on, its queue group, metadata, and its rate, errors and
  average time. An endpoint whose name matches a unary method of a Protobuf service in your
  [schema sources](/guide/protobuf) shows that method. An endpoint without a queue group is answered by
  every instance.
- **Last errors**: the last error of each endpoint, with the instance and version that reported it.
- **Instances**: each running copy with its version, rate, errors, start time, how long it took to
  answer (RTT) and metadata. **Raw JSON** shows its `$SRV.INFO` and `$SRV.STATS` replies, including
  any custom statistics the service adds.

**Call** opens [Request / Reply](/guide/request-reply) with the endpoint subject filled in. When the
endpoint matches a Protobuf method, its request and reply types are selected too. An endpoint subject
with wildcards needs its tokens filled in there.

## Permissions

Finding services needs permission to publish to `$SRV.INFO` and to receive the answers on a reply
inbox. Statistics need permission to publish to `$SRV.STATS`. Many accounts do not grant these, so the
page tells you what is missing instead of failing:

- **No access to services**: the page names the missing permission and stops refreshing. **Check
  again** retries once the permission has been granted. If your account receives replies on a private
  inbox, set its prefix in the [connection settings](/guide/connections) instead.
- **Statistics are hidden**: services and endpoints are listed without the statistics columns.
  Auto-refresh stops asking for statistics; **Refresh** checks the permission again.
- **No services found**: you have access, but nothing answered within 2 seconds. Services in other
  accounts are not visible from this connection.
