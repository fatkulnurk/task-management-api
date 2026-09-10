@echo off
setlocal
where mockgen >nul 2>nul || (echo mockgen not found in PATH & exit /b 1)

mockgen -source=internal/modules/auth/domain/repository.go -destination=internal/modules/auth/domain/mocks/mock_repository.go -package=mocks
mockgen -source=internal/modules/auth/domain/service.go -destination=internal/modules/auth/domain/mocks/mock_service.go -package=mocks
mockgen -source=internal/modules/teams/domain/repository.go -destination=internal/modules/teams/domain/mocks/mock_repository.go -package=mocks
mockgen -source=internal/modules/teams/domain/service.go -destination=internal/modules/teams/domain/mocks/mock_service.go -package=mocks
mockgen -source=internal/modules/tasks/domain/repository.go -destination=internal/modules/tasks/domain/mocks/mock_repository.go -package=mocks
mockgen -source=internal/modules/tasks/domain/service.go -destination=internal/modules/tasks/domain/mocks/mock_service.go -package=mocks
mockgen -source=internal/application/notification/notification.go -destination=internal/application/notification/mocks/mock_notification_service.go -package=mocks
mockgen -source=internal/application/token/token.go -destination=internal/application/token/mocks/mock_token_service.go -package=mocks

echo done
