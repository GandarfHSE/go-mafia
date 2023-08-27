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
![image](https://github.com/GandarfHSE/go-mafia/assets/80011710/c43d6828-7fdd-4c21-9936-8ab52e7fa6ec)


