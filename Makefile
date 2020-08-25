.PHONY: phony_jira_checker
phony_jira_checker:
	@docker run --rm  --network="host" -v "${PWD}/:/root" seriouscomp/phony \
		--address http://127.0.0.1:8888/jira-checker \
		--hmac e0e8b7f3b67db6837ead4aeabd14547be121d5de \
	  --event pull_request \
		--payload /root/examples/$(payload)

.PHONY: phony_checkmarx
phony_checkmarx:
	@docker run --rm  --network="host" -v "${PWD}/:/root" seriouscomp/phony \
		--address http://127.0.0.1:8888/checkmarx \
		--hmac e0e8b7f3b67db6837ead4aeabd14547be121d5de \
	  --event pull_request \
		--payload /root/examples/$(payload)

.PHONY: phony
phony:
	@docker run --rm  --network="host" -v "${PWD}/:/root" seriouscomp/phony \
		--address http://127.0.0.1:8888/$(route) \
		--hmac e0e8b7f3b67db6837ead4aeabd14547be121d5de \
	  --event $(event) \
		--payload /root/examples/$(payload)
