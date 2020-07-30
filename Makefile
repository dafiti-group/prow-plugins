.PHONY: phony
phony_jira_checker:
	@docker run --rm  --network="host" -v "${PWD}/:/root" seriouscomp/phony \
		--address http://127.0.0.1:8888/jira-checker \
		--hmac e0e8b7f3b67db6837ead4aeabd14547be121d5de \
	  --event pull_request \
		--payload /root/examples/$(payload)

