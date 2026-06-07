# Tailscale Connectivity Lab

A small Go learning lab for exploring Tailscale Funnel, DERP fallback behavior, source-address visibility, and Go-based Tailscale integration using `tailscale.com/tsnet`.

This project was built as an AI-assisted learning experiment to accelerate understanding of Tailscale's networking model, especially the areas relevant to Funnel, DERP relay infrastructure, Go services, and Linux network troubleshooting.

> **Redaction note:** This README is organized for sharing in a Git repository. Tailnet names, node names, public IPs, Tailscale keys, user login names, and profile URLs should be redacted or generalized before publishing publicly.

---

## Goals

The lab was designed to answer the following questions:

1. Can a Go service join a tailnet as an embedded Tailscale node using `tsnet`?
2. Can the same Go service be exposed privately over the tailnet and publicly through Funnel?
3. What request metadata does the backend see for private tailnet traffic versus public Funnel traffic?
4. How does `WhoIs` behave for:
   - private tailnet clients
   - tailnet-connected clients using the Funnel URL
   - non-tailnet public clients using the Funnel URL
5. Does Funnel expose the original public client IP address through normal HTTP headers?
6. Can the original public client IP and source port be recovered using PROXY protocol?
7. How does Tailscale behave when direct UDP connectivity is available, impaired, and then restored?
8. Can DERP fallback be reproduced in a controlled lab?

---

## Environment

The lab was run using the following environment:

| Component | Description |
|---|---|
| Development host | Windows 11 laptop |
| Local Linux environment | WSL Linux |
| Tailnet client | Tailscale installed on Windows |
| Go service | `tsnet`-based embedded Tailscale node |
| Test Linux node | Rocky Linux 8 VM |
| Tailnet DNS name | Redacted in this README; examples use `<tailnet>.ts.net` |
| Windows node | Redacted as `<windows-node>` |
| Linux node | Redacted as `<linux-node>` |
| Go service node | Redacted as `<tsnet-service-node>` |

The initial Go service was implemented using `tailscale.com/tsnet`, allowing the process itself to join the tailnet as a Tailscale node without relying on the host's existing `tailscaled` instance.

---

## Repository Components

Expected repository structure:

```text
.
├── README.md
├── go.mod
├── go.sum
├── main.go
└── cmd
    └── proxybackend
        └── main.go
```

Main components:

| Component | Purpose |
|---|---|
| `main.go` | Go `tsnet` HTTP service supporting private tailnet mode and Funnel mode |
| `cmd/proxybackend/main.go` | Go backend that parses PROXY protocol v2 before serving HTTP |
| `README.md` | Lab notes, commands, findings, and summaries |

---

## Service Endpoints

The `tsnet` service exposes the following endpoints:

| Endpoint | Purpose |
|---|---|
| `/healthz` | Basic health check |
| `/debug/request` | Returns request metadata as JSON |
| `/debug/whoami` | Uses Tailscale `WhoIs` to identify the caller |

The PROXY protocol backend exposes:

| Endpoint | Purpose |
|---|---|
| `/healthz` | Basic health check |
| `/debug/request` | Returns request metadata after parsing PROXY protocol v2 |

---

## Running the `tsnet` Service

### Private Tailnet Mode

```bash
export TS_AUTHKEY='tskey-auth-REDACTED'
export TSNET_HOSTNAME='<tsnet-service-node>'
export LAB_MODE='tailnet'

go run .
```

Private tailnet mode listens on port `8080` through the embedded Tailscale node.

Example test commands:

```powershell
curl.exe http://<tsnet-service-node>:8080/healthz
curl.exe http://<tsnet-service-node>:8080/debug/request
curl.exe http://<tsnet-service-node>:8080/debug/whoami
```

### Funnel Mode

```bash
export TS_AUTHKEY='tskey-auth-REDACTED'
export TSNET_HOSTNAME='<tsnet-service-node>'
export LAB_MODE='funnel'

go run .
```

Funnel mode exposes the service publicly through:

```text
https://<tsnet-service-node>.<tailnet>.ts.net/
```

Example test commands:

```powershell
curl.exe https://<tsnet-service-node>.<tailnet>.ts.net/healthz
curl.exe https://<tsnet-service-node>.<tailnet>.ts.net/debug/request
curl.exe https://<tsnet-service-node>.<tailnet>.ts.net/debug/whoami
```

---

# Experiment 1: Private Tailnet Access to a Go `tsnet` Service

## Purpose

Validate that a Go application can join the tailnet as an embedded Tailscale node and serve HTTP traffic privately over the tailnet.

## Test

From a Windows Tailscale node:

```powershell
curl.exe http://<tsnet-service-node>:8080/healthz
curl.exe http://<tsnet-service-node>:8080/debug/request
curl.exe http://<tsnet-service-node>:8080/debug/whoami
```

## Observed `/healthz` Result

```text
ok
```

## Observed `/debug/request` Result

```json
{
  "time": "2026-06-06T09:04:58.396103584-04:00",
  "method": "GET",
  "host": "<tsnet-service-node>:8080",
  "url": "/debug/request",
  "remote_addr": "100.x.y.z:56694",
  "user_agent": "curl/8.13.0",
  "headers": {
    "Accept": ["*/*"],
    "User-Agent": ["curl/8.13.0"]
  },
  "tls": false
}
```

## Observed `/debug/whoami` Result

The private tailnet request was identified as coming from the Windows Tailscale node:

```text
Node name: <windows-node>.<tailnet>.ts.net.
Computed name: <windows-node>
Computed name with host: <windows-node> (<windows-hostname>)
OS: windows
Tailscale IPv4: 100.x.y.z/32
Tailscale IPv6: fd7a:115c:a1e0::.../128
Home DERP: 21
Online: true
```

Sensitive values such as node keys, disco keys, user login names, and profile URLs were intentionally omitted.

## Finding

The Go `tsnet` service successfully joined the tailnet and served private HTTP traffic. In private tailnet mode:

- the request used plain HTTP
- `tls` was `false`
- the backend saw the caller's Tailscale IP
- `WhoIs` resolved the caller to the Windows Tailscale node identity

---

# Experiment 2: Funnel Access from a Tailnet-Connected Client

## Purpose

Test the same Go service through the public Funnel URL while the client is also connected to the tailnet.

## Test

From the Windows Tailscale node:

```powershell
curl.exe https://<tsnet-service-node>.<tailnet>.ts.net/healthz
curl.exe https://<tsnet-service-node>.<tailnet>.ts.net/debug/request
curl.exe https://<tsnet-service-node>.<tailnet>.ts.net/debug/whoami
```

## Observed `/healthz` Result

```text
ok
```

## Observed `/debug/request` Result

```json
{
  "time": "2026-06-06T09:14:46.059002504-04:00",
  "method": "GET",
  "host": "<tsnet-service-node>.<tailnet>.ts.net",
  "url": "/debug/request",
  "remote_addr": "100.x.y.z:52943",
  "user_agent": "curl/8.13.0",
  "headers": {
    "Accept": ["*/*"],
    "User-Agent": ["curl/8.13.0"]
  },
  "tls": true
}
```

## Observed `/debug/whoami` Result

The request was still identified as coming from the Windows Tailscale node:

```text
Node name: <windows-node>.<tailnet>.ts.net.
Computed name: <windows-node>
Computed name with host: <windows-node> (<windows-hostname>)
OS: windows
Tailscale IPv4: 100.x.y.z/32
Home DERP: 21
Online: true
```

## Finding

Although the request used the public Funnel URL and HTTPS, the caller was still a tailnet-connected device. As a result:

- `tls` was `true`
- the `Host` header used the Funnel hostname
- the remote address was still the Windows node's Tailscale IP
- `WhoIs` resolved the caller as the Windows Tailscale node

This showed that using a Funnel URL from a tailnet-connected client does not simulate an anonymous public internet client.

---

# Experiment 3: Funnel Access from a Non-Tailnet Client

## Purpose

Test the public Funnel URL from a machine that was not itself a tailnet node.

## Test

From a non-tailnet Linux client:

```bash
curl https://<tsnet-service-node>.<tailnet>.ts.net/healthz
curl https://<tsnet-service-node>.<tailnet>.ts.net/debug/request
curl https://<tsnet-service-node>.<tailnet>.ts.net/debug/whoami
```

## Observed `/healthz` Result

```text
ok
```

## Observed `/debug/request` Result

```json
{
  "time": "2026-06-06T11:36:59.697445471-04:00",
  "method": "GET",
  "host": "<tsnet-service-node>.<tailnet>.ts.net",
  "url": "/debug/request",
  "remote_addr": "[fd7a:115c:a1e0::...]:21625",
  "user_agent": "curl/7.61.1",
  "headers": {
    "Accept": ["*/*"],
    "User-Agent": ["curl/7.61.1"]
  },
  "tls": true
}
```

## Observed `/debug/whoami` Result

The public Funnel request was not identified as the original public internet client. Instead, `WhoIs` identified the caller as a special Tailscale ingress node:

```text
Computed name: funnel-ingress-node
Hostname: funnel-ingress-node
Sharee node: true
Tags: tag:ingress
User profile: Tagged Devices
Home DERP: 1
Ingress capability: https://tailscale.com/cap/ingress
Online: true
Jailed: true
```

Sensitive values such as node keys and disco keys were intentionally omitted.

## Finding

Public Funnel traffic is proxied into the tailnet through a Tailscale-managed ingress identity.

From the Go service's point of view, the caller was not the original public client. It was the Tailscale Funnel ingress node.

```text
Public internet client
        |
        | HTTPS
        v
Tailscale Funnel ingress
        |
        | Tailscale transport
        v
Go tsnet service
```

This is useful for understanding the difference between:

- application-layer public HTTP identity
- Tailscale node identity
- reverse-proxy or ingress identity
- source address visibility at the backend service

---

# Experiment 4: Preserving Public Client Source Information with PROXY Protocol

## Purpose

Determine whether original public client source IP and source port information can be surfaced to a backend service.

The previous Funnel test showed that a non-tailnet client appears to the `tsnet` backend as `funnel-ingress-node`. Normal HTTP headers did not include the original public client IP.

This experiment tested Tailscale CLI Funnel with PROXY protocol v2.

## Test Architecture

```text
Public non-tailnet client
        |
        | HTTPS
        v
Tailscale Funnel
        |
        | TLS terminated by Tailscale
        | TCP forwarded with PROXY protocol v2
        v
Go PROXY-protocol-aware backend
```

## Backend Setup

A separate Go backend was created at:

```text
cmd/proxybackend/main.go
```

The backend listens on local TCP port `9899`, parses the PROXY protocol v2 header if present, and then serves HTTP.

Run the backend:

```bash
go run ./cmd/proxybackend
```

Confirm direct local access:

```bash
curl http://127.0.0.1:9899/healthz
curl http://127.0.0.1:9899/debug/request
```

Expected direct local behavior:

```text
remote_addr = 127.0.0.1:<port>
tls = false
```

## Funnel CLI Setup

The Tailscale CLI Funnel TCP forwarder was placed in front of the backend:

```bash
tailscale funnel --proxy-protocol=2 --tls-terminated-tcp=443 tcp://127.0.0.1:9899
```

Example Funnel output:

```text
Available on the internet:

|-- tcp://<windows-node>.<tailnet>.ts.net:443 (TLS terminated, PROXY protocol v2)
|-- tcp://100.x.y.z:443
|-- tcp://[fd7a:115c:a1e0::...]:443
|--> tcp://127.0.0.1:9899

Press Ctrl+C to exit.
```

## Test from Non-Tailnet Client

```bash
curl https://<windows-node>.<tailnet>.ts.net/healthz
curl https://<windows-node>.<tailnet>.ts.net/debug/request
```

## Observed Result

The backend successfully recovered the original public client IP address and source port from the PROXY protocol header:

```text
remote_addr = <public-client-ip>:<source-port>
tls = false
```

The backend saw `tls=false` because TLS was terminated by Tailscale before the TCP connection was forwarded to the backend.

## Finding

There are two different backend visibility models:

| Funnel Path | Backend Sees |
|---|---|
| `tsnet.ListenFunnel` basic HTTP service | Tailscale Funnel ingress node/address |
| `tailscale funnel --proxy-protocol=2` to PROXY-aware backend | Original public client IP/port from PROXY protocol |

PROXY protocol is the mechanism for surfacing original public client source information to the backend. Normal Funnel HTTP request headers did not provide this information in the `tsnet.ListenFunnel` test.

## Security Note

A backend should only trust PROXY protocol information when the connection comes from a trusted local forwarder or trusted proxy path.

If a service accepts arbitrary internet traffic directly and blindly trusts PROXY protocol headers, a client could spoof source information.

In this lab, the backend was only exposed through the local Tailscale Funnel forwarder, so trusting the PROXY protocol header was appropriate for the experiment.

## Cleanup

Check current Funnel state:

```bash
tailscale funnel status
```

Clear the Funnel config when done:

```bash
tailscale funnel reset
```

---

# Experiment 5: DERP Fallback and Direct Path Recovery

## Purpose

Test how Tailscale behaves when direct UDP connectivity is available, then intentionally impaired, then restored.

## Topology

```text
Windows laptop / <windows-node>
        |
        | Same LAN: 192.168.2.0/24
        |
Rocky Linux 8 VM / <linux-node>
```

## Baseline

Before changing firewall rules, `tailscale ping` initially used DERP and then converged to a direct LAN path:

```text
pong from <linux-node> (100.x.y.z) via DERP(tor) in 20ms
pong from <linux-node> (100.x.y.z) via DERP(tor) in 21ms
pong from <linux-node> (100.x.y.z) via DERP(tor) in 19ms
pong from <linux-node> (100.x.y.z) via DERP(tor) in 19ms
pong from <linux-node> (100.x.y.z) via 192.168.2.83:41641 in 1ms
```

Subsequent pings used the direct path immediately:

```text
pong from <linux-node> (100.x.y.z) via 192.168.2.83:41641 in 1ms
```

From the Linux side, the direct path was also confirmed:

```text
pong from <windows-node> (100.x.y.z) via 192.168.2.63:41641 in 1ms
```

## UDP Block

UDP was then blocked on the Linux test node:

```bash
sudo iptables -I OUTPUT 1 -p udp -j REJECT
sudo iptables -I INPUT 1 -p udp -j REJECT
```

After blocking UDP, Tailscale could no longer establish the direct WireGuard path. `tailscale ping` remained on DERP:

```text
pong from <linux-node> (100.x.y.z) via DERP(tor) in 19ms
pong from <linux-node> (100.x.y.z) via DERP(tor) in 19ms
pong from <linux-node> (100.x.y.z) via DERP(tor) in 19ms
pong from <linux-node> (100.x.y.z) via DERP(tor) in 21ms
direct connection not established
```

The Linux side showed the same result:

```text
pong from <windows-node> (100.x.y.z) via DERP(tor) in 20ms
pong from <windows-node> (100.x.y.z) via DERP(tor) in 20ms
pong from <windows-node> (100.x.y.z) via DERP(tor) in 19ms
direct connection not established
```

`tailscale status` also showed the peer path as relayed:

```text
<linux-node>    active; offers exit node; relay "tor", tx 300 rx 188
<windows-node>  active; relay "tor", tx 348 rx 372
```

## Recovery

The UDP reject rules were removed from the Linux test node:

```bash
sudo iptables -D OUTPUT -p udp -j REJECT
sudo iptables -D INPUT -p udp -j REJECT
```

After removing the rules, the direct path recovered immediately.

From the Linux node:

```text
[root@<linux-node> ~]# tailscale ping <windows-node>
pong from <windows-node> (100.x.y.z) via 192.168.2.63:41641 in 1ms
```

`tailscale status` on the Linux node showed the Windows peer using a direct LAN path:

```text
100.x.y.z  <windows-node>  user@example.com  windows  active; direct 192.168.2.63:41641, tx 632 rx 808
```

From Windows:

```text
PS C:\Users\user\Downloads> tailscale ping <linux-node>
pong from <linux-node> (100.x.y.z) via 192.168.2.83:41641 in 1ms
```

`tailscale status` on Windows also showed the Linux peer using a direct LAN path:

```text
100.x.y.z  <linux-node>  user@example.com  linux  active; offers exit node; direct 192.168.2.83:41641, tx 888 rx 568
```

## Findings

This experiment demonstrated Tailscale's direct/relay path selection behavior in a controlled LAN environment:

| State | Observed Path | Latency |
|---|---|---:|
| Baseline during initial discovery | DERP relay via `tor` region | ~19-21 ms |
| Baseline after discovery | Direct LAN path | ~1 ms |
| UDP blocked | DERP relay via `tor` region | ~13-30 ms |
| UDP restored | Direct LAN path | ~1 ms |

The experiment showed that Tailscale can:

1. use DERP temporarily while attempting path discovery
2. switch to a direct WireGuard UDP path when available
3. fall back to DERP when UDP is blocked
4. recover the direct path once UDP connectivity is restored

```text
Normal state:
  <windows-node> <---- direct WireGuard UDP ----> <linux-node>

UDP impaired:
  <windows-node> <---- DERP relay ----> <linux-node>

UDP restored:
  <windows-node> <---- direct WireGuard UDP ----> <linux-node>
```

DERP provides reachability when direct UDP connectivity fails. It does not replace WireGuard encryption; the traffic remains encrypted between the Tailscale nodes.

For this environment, `tor` refers to the Tailscale DERP region code for Toronto.

---

# Cross-Experiment Summary

## Request Path Comparison

| Experiment | Client Type | Entry Point | Backend Sees | Identity Result |
|---|---|---|---|---|
| Private tailnet | Tailnet node | `http://<tsnet-service-node>:8080` | Client Tailscale IP | Client Tailscale node |
| Funnel from tailnet client | Tailnet node | `https://<tsnet-service-node>.<tailnet>.ts.net` | Client Tailscale IP | Client Tailscale node |
| Funnel from non-tailnet client | Public internet client | `https://<tsnet-service-node>.<tailnet>.ts.net` | Funnel ingress address | `funnel-ingress-node` |
| Funnel with PROXY protocol | Public internet client | `https://<windows-node>.<tailnet>.ts.net` | Original public client IP/port | Provided by PROXY protocol, not `WhoIs` |
| DERP fallback | Tailnet node | Tailscale peer connection | Relay path shown in `tailscale ping/status` | Peer remains same Tailscale node |

## Key Findings

1. A Go application can use `tsnet` to join a tailnet and serve traffic as an embedded Tailscale node.
2. The same service can expose private tailnet endpoints and public Funnel endpoints.
3. A tailnet-connected client using a Funnel URL can still be identified as its Tailscale node through `WhoIs`.
4. A non-tailnet public client using Funnel appears to the backend as a Tailscale-managed `funnel-ingress-node`.
5. Normal HTTP headers did not expose the original public client IP in the basic `tsnet.ListenFunnel` test.
6. Tailscale CLI Funnel with PROXY protocol v2 allowed a backend to recover the original public client IP and source port.
7. Tailscale uses DERP as a fallback relay when direct UDP connectivity cannot be established.
8. Tailscale can return from DERP relay mode to direct peer-to-peer WireGuard connectivity once UDP connectivity is restored.

---

# Troubleshooting Notes

## Stale Funnel State / Duplicate `tsnet` Machine Entry

During testing, Funnel DNS and TCP/443 were reachable, but TLS closed before presenting a certificate.

Observed symptom:

```text
Funnel hostname resolves and TCP/443 connects, but TLS closes before certificate delivery.
```

Suspected flow:

```mermaid
flowchart TD
    A["Stale tsnet node"]
    B["Old Funnel mapping"]
    C["Client reaches ingress"]
    D["Routing/TLS cannot complete"]
    E["TLS closes before certificate"]

    A --> B --> C --> D --> E
```

Failure sequence:

1. An old `tsnet` node identity or machine entry still existed.
2. The Funnel DNS/control-plane mapping still pointed at that stale identity.
3. An external client reached the Tailscale Funnel ingress.
4. The ingress could not complete routing/TLS to a valid active node.
5. The TLS connection closed before certificate delivery.

Resolution:

1. Delete the stale `<tsnet-service-node>` machine from the Tailscale admin console.
2. Restart the `tsnet` app.
3. Allow the node identity and Funnel routing state to be recreated.

---

# Security Notes

The following files and values should not be committed:

- `tsnet-state/`
- `.env`
- Tailscale auth keys
- node keys
- disco keys
- personal login names
- profile URLs
- raw `WhoIs` JSON containing key material
- unredacted public IP addresses, unless intentionally included

Recommended `.gitignore`:

```gitignore
tsnet-state/
.env
*.log
```

Before committing changes:

```bash
grep -R "tskey-" .
grep -R "nodekey:" .
grep -R "discokey:" .
git status
```

---

# Future Work

Possible next experiments:

1. Build and inspect the open-source `derper` command.
   - Review how DERP differs from a normal HTTP reverse proxy.
   - Document the relationship between DERP, WireGuard packets, DISCO packets, and fallback relay behavior.

2. Add structured logging to the Go services.
   - Timestamp
   - Mode
   - Host
   - Remote address
   - TLS state
   - User-Agent
   - `WhoIs` result, if available

3. Add a small test script.
   - Run each endpoint.
   - Capture outputs.
   - Redact sensitive values.
   - Generate repeatable experiment logs.

4. Compare `tsnet.ListenFunnel` with Tailscale CLI Funnel behavior.
   - Document feature differences.
   - Investigate whether PROXY protocol support exists or could be added for embedded `tsnet` Funnel use cases.

---

# Bug Fix Ideas / Feature Requests

These are rough ideas that came out of the experiment.

## Tailnet Naming

It would be useful to have a way to rename a personal tailnet rather than relying on a randomly assigned name.

Potential considerations:

- guard against existing tailnet names
- prevent confusingly similar names
- define ownership and migration behavior for existing DNS names

## Stale Funnel State Handling

The stale `tsnet` node issue may suggest an opportunity for improved diagnostics or cleanup flows.

Possible improvements:

- clearer CLI or admin-console warning when Funnel points at a stale machine identity
- easier way to reset Funnel state for a deleted or recreated `tsnet` node
- clearer error when TLS closes before certificate delivery because the ingress cannot route to the active node

---

# Resume / Interview Summary

This lab demonstrates hands-on learning with Tailscale's Go networking stack:

> Built a Go-based Tailscale connectivity lab using `tsnet` to expose an embedded HTTP service over both private tailnet and public Funnel paths. Compared request metadata, TLS behavior, and Tailscale `WhoIs` identity results between direct tailnet access and Funnel access.

Additional summary:

> Extended the lab with a PROXY protocol backend and Tailscale CLI Funnel TCP forwarding. The basic `tsnet.ListenFunnel` path exposed public non-tailnet requests as coming from Tailscale's `funnel-ingress-node`, while the CLI Funnel PROXY protocol path allowed the backend to recover the original public client IP and source port.

DERP summary:

> Reproduced Tailscale DERP fallback behavior by comparing direct LAN connectivity, intentionally blocking UDP on a Linux test node, observing DERP relay fallback through `tailscale ping` and `tailscale status`, and then confirming recovery back to a direct WireGuard UDP path after restoring UDP connectivity.

Overall:

> Used AI-assisted learning to accelerate code scaffolding, experiment design, repo organization, and interpretation of results while validating the behavior directly with working Go code, Tailscale CLI output, and controlled Linux firewall changes.
