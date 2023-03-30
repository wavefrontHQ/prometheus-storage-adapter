# Testing a Prometheus Storage Adapter for Operations for Applications

1. Update the proxy with YOUR_API_TOKEN and YOUR_CLUSTER in hack/test/6-wavefront-proxy.yaml.
2. Create a namespace by running: `kubectl create namespace monitoring`.
3. Apply all yamls within hack/test by running:
   `kubectl apply -f hack/test/`
4. The Prometheus metrics should now be available under the metric name with a prefix "prometheus.".
