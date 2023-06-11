### Client-server mafia app

Чтобы поднять сервер:
```bash
docker-compose build
docker-compose up
```
У меня плчему-то были проблемы с таймаутом, но с `COMPOSE_HTTP_TIMEOUT=300` вроде работает =)

Чтобы поднять клиент:
```bash
cd app/client
go build
./client
```
