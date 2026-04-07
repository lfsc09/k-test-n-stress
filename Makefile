.PHONY: install-hooks
install-hooks:
	git config core.hooksPath .githooks
	chmod +x .githooks/post-commit