---
title: Services
description: Find NATS Micro services, see their endpoints, instances and statistics, and call an endpoint from Request / Reply.
---

# Services

**Services** in the sidebar lists the services built with the NATS Micro framework (in Go, JavaScript,
Python, Rust and other clients) that answer on the current connection. Natscope asks them on the
`$SRV` subjects, the same way `nats micro ls` does.

## The list

Each service shows its name and description, its versions, how many instances run, its endpoints, and
the totals of requests, errors and average processing time across all endpoints.

The list refreshes every 5 seconds while **Auto-refresh** is on. **Refresh** asks again at once.
Type in the filter to narrow the list by service name, description, endpoint name or subject.

## A service

Click a service to see:

- **Endpoints**: the subject each endpoint listens on, its queue group, requests, errors and average
  processing time. An endpoint whose name matches a unary method of a Protobuf service in your
  [schema sources](/guide/protobuf) shows that method.
- The last error each endpoint reported.
- **Instances**: the running copies of the service, with version, start time and metadata.

**Call** opens [Request / Reply](/guide/request-reply) with the endpoint subject filled in. When the
endpoint matches a Protobuf method, its request and reply types are selected too.

## Permissions

Finding services needs permission to publish to `$SRV.INFO` and to receive the answers on a reply
inbox. Statistics need permission to publish to `$SRV.STATS`. Many accounts do not grant these, so the
page tells you what is missing instead of failing:

- **No access to services**: the page names the missing permission and stops refreshing. **Check
  again** retries once the permission has been granted.
- **Statistics are hidden**: services and endpoints are listed, and the statistics columns show `—`.
  Auto-refresh stops asking for statistics; **Refresh** checks the permission again.
- **No services found**: you have access, but nothing answered.
