HELM := $(CURDIR)/../bin/helm
CHART_NAME := argocd
NAMESPACE := apis
RELEASE_NAME := cat-app
VALUES_FILE := values.yaml

.PHONY: install upgrade uninstall lint

install: uninstall
	$(HELM) install $(RELEASE_NAME) . \
		--namespace $(NAMESPACE) \
		--create-namespace \
		-f $(VALUES_FILE)

upgrade:
	$(HELM) upgrade $(RELEASE_NAME) . \
		--namespace $(NAMESPACE) \
		-f $(VALUES_FILE)

uninstall:
	-$(HELM) uninstall $(RELEASE_NAME) \
		--namespace $(NAMESPACE) || true

lint:
	$(HELM) lint .
