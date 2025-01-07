# Финальный проект 1 семестра (простой (первый) уровень сложности)

REST API сервис для загрузки и выгрузки данных о ценах

## Требования к системе

- **Аппаратные требования**: 2ГБ ОЗУ и 5 ГБ дискового пространства или выше
- **Операционная система**: Linux (Ubuntu 22.04 или выше)
- **СУБД**: PostgreSQL (16.2 или выше)

## Установка и запуск

1. Установите PostgreSQL и настройте базу данных:

```bash
sudo apt update
sudo apt install postgresql
sudo su - postgres
psql
CREATE DATABASE "project-sem-1";
CREATE USER validator WITH PASSWORD 'val1dat0r';
\c project-sem-1
ALTER DATABASE "project-sem-1" OWNER TO validator;
ALTER SCHEMA "public" OWNER TO validator;
\q
```

2. Установите Go (версия 1.23 или выше)

3. Склонируйте репозиторий проекта:

```bash
git clone git@github.com:ClarkeFeed/itmo-devops-sem1-project-template.git
cd itmo-devops-sem1-project-template
```

4. Запустите скрипт подготовки:

```bash
bash ./scripts/prepare.sh
```

5. Запустите сервер локально:

```bash
bash ./scripts/run.sh
```

## Тестирование

1. Запустите тесты API-запросов с помощью скрипта:

```bash
sh ./scripts/tests.sh 1
```

2. Скрипт `tests.sh` проверяет:

- Корректность добавления данных через `POST /api/v0/prices`
- Корректность выгрузки данных через `GET /api/v0/prices`

### Пример успешного выполнения тестов:

- **POST запрос** загружает данные из архива и возвращает JSON с метриками
```json
{
  "total_items": 100,
  "total_categories": 15,
  "total_price": 100000
}
```
**Пример запроса:**

```bash
curl -X POST -F "file=@sample_data.zip" http://localhost:8080/api/v0/prices
```

- **GET запрос** возвращает zip-архив с корректным содержимым файла `data.csv`
**Пример запроса:**

```bash
curl -X GET -o ./sample_data/output.zip http://localhost:8080/api/v0/prices
```

## Контакт

В случае вопросов можно обращаться:

- **Telegram**: [@raaaaagh](https://t.me/raaaaagh)