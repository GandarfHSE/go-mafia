### Client-server mafia app

Чтобы поднять сервер:
```bash
docker-compose build
docker-compose up
```

Чтобы поднять клиент:
```bash
cd app/client
go build
./client
```
Все команды в клиенте начинаются с `!`. Доступные команды можно увидеть с помощью команды `!help`.
![App screenshot](image.png)
