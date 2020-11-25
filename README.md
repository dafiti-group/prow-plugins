[![Docker Repository on Quay](https://quay.io/repository/dafiti/prow-plugins/status "Docker Repository on Quay")](https://quay.io/repository/dafiti/prow-plugins)

# Prow Plugins

A collection of standalone plugins that use prow as a framework "technically" this plugins don't need a running instace of prow to work, it does not connect directly to prow in any way but you need prow for some behavious to work like reacting to a label on github for example.

## Running locally

To run this locally you are going to need:
- [tilt](https://docs.tilt.dev/install.html) for local running
- [kind or any other kubernetes cluster](https://kind.sigs.k8s.io/docs/user/quick-start/)
- [phony](https://hub.docker.com/repository/docker/seriouscomp/phony/general) or curl
- [Github Token](https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token) you need this to communicate with github api

Before running you need to setup the github access token, assuming you have it in you environment variables running `echo "oauth=$GITHUB_ACCESS_TOKEN" > config/secrets/env` is all you need

The `hmac` is the token used for a basic authentication, the fake one used for this example is `e0e8b7f3b67db6837ead4aeabd14547be121d5de`

Assuming you have a k8s cluster running execute `tilt up` after a few seconds the application should be up and running, you can execute phony or a curl to make a request
```
docker run --rm  --network="host" -v "$PWD/:/root" seriouscomp/phony \
  --address http://127.0.0.1:8888/jira-checker \
  --hmac e0e8b7f3b67db6837ead4aeabd14547be121d5de \
  --event pull_request \
  --payload /root/examples/push.json
```

The contents inside example are github request, you can check the webhook fireds [here](https://github.com/organizations/dafiti-group/settings/hooks/224575357)

## Testing

## Deploy

although the config folder contains is a valid k8s manifest with kustomize, this configurations are for locall running only, you can deploy this plugins using them but it's recomended to deploy allong side prow itself with [prow chart](https://github.com/dafiti-group/charts/tree/master/charts/gfg-prow)

## Reference

- [Prow external plugins](https://github.com/kubernetes/test-infra/tree/master/prow/external-plugins)
- [Prow plugins](https://github.com/kubernetes/test-infra/tree/master/prow/plugins)
- [Release Blocker Plugin](https://github.com/davidvossel/release-blocker-plugin)
