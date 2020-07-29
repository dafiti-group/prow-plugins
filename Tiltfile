# Deploy: tell Tilt what YAML to deploy
k8s_yaml(kustomize('config'))

# Build: tell Tilt what images to build from which directories
docker_build('quay.io/dafiti/prow-plugins', '')

# Watch: tell Tilt how to connect locally (optional)
k8s_resource(
  'prow-plugins',
  port_forwards=8888
)