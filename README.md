Описание: небольшое gRPC приложение, реализующее авторизацию и взаимодействие с JWT-токенами.
* gRPC/cmd содержит точку входа в приложение
* gRPC/gRPC содержит .proto файл и репозиторий genGO с кодогенерацией
* gRPC/internal содержит репозитории с пакетами config, repo, services
* gRPC/pgk содержит пакеты для валидации, логгирования, формирование JWT-токенов

Команда для protobuff-генерации:
    protoc --go_out=grpc/genGo  --go-grpc_out=grpc/genGo  grpc/proto/auth.proto

Команды для создания мок-репозитория:
    cd .\internal\repo\
    mockery --name Repository