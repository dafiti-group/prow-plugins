[![Docker Repository on Quay](https://quay.io/repository/dafiti/prow-plugins/status "Docker Repository on Quay")](https://quay.io/repository/dafiti/prow-plugins)

# prow-plugins

## Running

```
go run *.go \
  --github-token-path ./fake-token \
  --hmac-secret-file ./fake-token \
  --prow-url "<deck-url>" \
  --config-path config.yaml
```

## Local Testing
Install [phony](https://github.com/kubernetes/test-infra/tree/master/prow/cmd/phony)

```
phony --address http://127.0.0.1:8888 \
  --hmac 123 \
  --event issue_comment
```

## Reference

- [Prow external plugins](https://github.com/kubernetes/test-infra/tree/master/prow/external-plugins)
- [Prow plugins](https://github.com/kubernetes/test-infra/tree/master/prow/plugins)
- [Release Blocker Plugin](https://github.com/davidvossel/release-blocker-plugin)
