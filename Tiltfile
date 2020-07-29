# Load extensions
load('ext://secret', 'secret_create_generic')

# Deploy: tell Tilt what YAML to deploy
k8s_yaml(kustomize('config'))

# Create the secrets
secret_create_generic(
  name = 'github-token',
  namespace = 'prow',
  from_file= 'oauth=YjY4OWUxNjc2MzI2MzI1OGQ1YzgyM2ZmMDAyZTczaGFrc3VkamZ1Y2oK'
)

# Build: tell Tilt what images to build from which directories
docker_build('quay.io/dafiti/prow-plugins', '')

# Watch: tell Tilt how to connect locally (optional)
k8s_resource(
  'prow-plugins',
  port_forwards=8888
)