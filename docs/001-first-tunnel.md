# Publish your first Service

The sample exposes a demo HTTP server through rstream. Use an empty
`rstream-demo` namespace for the exercise. A published endpoint is
reachable from outside the cluster, so use the sample's disposable content.

## 1. Start the sample Service

After [installing the Operator](000-getting-started.md), save this manifest as
`http-server.yaml`. It creates the demo namespace, Deployment, and Service.
The server exposes only the fixed test response.

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: rstream-demo
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: http-server
  namespace: rstream-demo
spec:
  replicas: 1
  selector:
    matchLabels:
      app: http-server
  template:
    metadata:
      labels:
        app: http-server
    spec:
      automountServiceAccountToken: false
      containers:
        - name: http-server
          image: python:3.12-alpine
          command:
            - sh
            - -ec
            - |
              mkdir -p /tmp/rstream-demo
              printf 'hello from rstream Operator\n' > /tmp/rstream-demo/index.html
              exec python -m http.server 8080 --bind 0.0.0.0 --directory /tmp/rstream-demo
          ports:
            - containerPort: 8080
              name: http
---
apiVersion: v1
kind: Service
metadata:
  name: http-server
  namespace: rstream-demo
spec:
  selector:
    app: http-server
  ports:
    - name: http
      port: 8080
      targetPort: 8080
```

```sh
kubectl apply -f http-server.yaml
kubectl -n rstream-demo rollout status deployment/http-server
```

## 2. Configure the connection

| Connection | What to prepare |
| --- | --- |
| Hosted rstream project | The project endpoint from the rstream Dashboard and a project token authorized to read the project and create tunnels. The manager resolves the Engine through `https://rstream.io`. |
| Self-hosted Engine | The Engine address as `host:port` and a token the Engine accepts. To publish the sample, the Engine also needs routing, DNS, and certificate coverage for its published tunnel hostnames. |

For a hosted project, open the workspace's Tokens page in the Dashboard.
Create an access token scoped to that project with
`account.projects.read-only` for endpoint resolution and
`tunnels.tunnels.create-delete` for the agent. Use a disposable project
that permits anonymous published HTTP for this walkthrough.
See [rstream Tokens](https://rstream.io/docs/security/tokens).

Save the token by itself in a local file with restricted access. Use that
file's path in the `--from-file` argument.

```sh
kubectl -n rstream-demo create secret generic rstream-credentials \
  --from-file=token=/path/to/token.txt
```

Save this manifest as `connection.yaml` and replace `YOUR_PROJECT_ENDPOINT` with
the project endpoint identifier from the Dashboard. For a direct connection,
replace the entire `projectEndpoint` line with `engine: engine.example.com:443`,
using your Engine address.

```yaml
apiVersion: tunnels.rstream.io/v1alpha1
kind: RstreamConnection
metadata:
  name: default
  namespace: rstream-demo
spec:
  projectEndpoint: "YOUR_PROJECT_ENDPOINT"
  tokenSecretRef:
    name: rstream-credentials
    key: token
```

```sh
kubectl apply -f connection.yaml
kubectl -n rstream-demo wait --for=condition=Ready rstreamconnection/default --timeout=120s
kubectl -n rstream-demo get rstreamconnection default
```

Connection readiness confirms that the manager could read the referenced
credential and resolve the settings. The agent connects to the Engine in
the next step. See the [API reference](003-api-reference.md) for mTLS and
custom certificate trust.

## 3. Publish the Service

Save the tunnel as `tunnel.yaml`. The target `http` is the Service port name.

```yaml
apiVersion: tunnels.rstream.io/v1alpha1
kind: RstreamTunnel
metadata:
  name: web
  namespace: rstream-demo
spec:
  connectionRef:
    name: default
  target:
    service:
      name: http-server
      port: http
  publish: true
  protocol: http
  http:
    version: http/1.1
```

```sh
kubectl apply -f tunnel.yaml
kubectl -n rstream-demo wait --for=condition=AgentReady rstreamtunnel/web --timeout=120s
kubectl -n rstream-demo wait --for=condition=Ready rstreamtunnel/web --timeout=120s
kubectl -n rstream-demo get rstreamtunnel web
```

## 4. Verify HTTP forwarding

```sh
address="$(kubectl -n rstream-demo get rstreamtunnel web -o jsonpath='{.status.forwardingAddress}')"
curl --fail --show-error --include "$address"
```

Expect HTTP 200 and the body `hello from rstream Operator`. This confirms
that a request sent to the published address reaches the Service. Use [diagnostics](005-troubleshooting.md)
if readiness or HTTP forwarding fails.

For an existing application, create the connection and Secret in its
namespace and set the target to its Service port. Decide whether the
application needs a public endpoint. A private tunnel uses `publish: false`
and can display a private `rstrm://` address. Use an authenticated rstream
client to reach it. See [tunnel policy and HTTP access controls](003-api-reference.md).

## 5. Remove the sample

Delete the tunnel first so the manager can remove its agent and the forwarding
session can close. Leave the manager running during tunnel deletion.

```sh
kubectl -n rstream-demo delete rstreamtunnel web
kubectl delete namespace rstream-demo
```
