# Testing Prometheus Storage Adapter for Wavefront

1. Update proxy with YOUR_API_TOKEN and YOUR_CLUSTER in hack/test/6-wavefront-proxy.yaml.
2. Create namespace by `kubectl create namespace monitoring`
3. Apply all yamls within hack/test by
   `kubectl apply -f hack/test/`
4. The prometheus metrics should now be available under the metric name with prefix "prometheus."