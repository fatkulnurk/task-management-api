MOCKGEN ?= mockgen

.PHONY: mocks mocks-auth mocks-teams mocks-tasks mocks-notification mocks-token test test-verbose test-cover test-race test-integration jwt-secret

mocks: mocks-auth mocks-teams mocks-tasks mocks-notification mocks-token

mocks-auth:
	$(MOCKGEN) -source=internal/modules/auth/domain/repository.go \
		-destination=internal/modules/auth/domain/mocks/mock_repository.go \
		-package=mocks
	$(MOCKGEN) -source=internal/modules/auth/domain/service.go \
		-destination=internal/modules/auth/domain/mocks/mock_service.go \
		-package=mocks

mocks-teams:
	$(MOCKGEN) -source=internal/modules/teams/domain/repository.go \
		-destination=internal/modules/teams/domain/mocks/mock_repository.go \
		-package=mocks
	$(MOCKGEN) -source=internal/modules/teams/domain/service.go \
		-destination=internal/modules/teams/domain/mocks/mock_service.go \
		-package=mocks

mocks-tasks:
	$(MOCKGEN) -source=internal/modules/tasks/domain/repository.go \
		-destination=internal/modules/tasks/domain/mocks/mock_repository.go \
		-package=mocks
	$(MOCKGEN) -source=internal/modules/tasks/domain/service.go \
		-destination=internal/modules/tasks/domain/mocks/mock_service.go \
		-package=mocks

mocks-notification:
	$(MOCKGEN) -source=internal/application/notification/notification.go \
		-destination=internal/application/notification/mocks/mock_notification_service.go \
		-package=mocks

mocks-token:
	$(MOCKGEN) -source=internal/application/token/token.go \
		-destination=internal/application/token/mocks/mock_token_service.go \
		-package=mocks

test:
	go test ./...

test-verbose:
	go test -v ./...

test-cover:
	go test "-coverprofile=coverage.out" ./...
	go tool cover -func=coverage.out

test-race:
	docker run --rm -v "$(CURDIR):/src" -w /src -v taskmanagement-gomod:/go/pkg/mod golang:1.27 go test -race ./...

test-integration:
	docker run --rm --network taskmanagement_default -e "TEST_DATABASE_URL=root:rootpassword@tcp(mysql:3306)/?multiStatements=true" -v "$(CURDIR):/src" -w /src -v taskmanagement-gomod:/go/pkg/mod golang:1.27 go test -race -tags=integration ./internal/modules/tasks/repository/...

jwt-secret:
	powershell -NoProfile -ExecutionPolicy Bypass -File scripts/generate-jwt-secret.ps1
