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

Директория `sample_data` - это пример директории, которая является разархивированной версией файла `sample_data.zip`

Какие тесты проходит приложение? Можно предоставить команду для тестов или описание тестов со скришотами/видео.

## Контакт

К кому можно обращаться в случае вопросов?
