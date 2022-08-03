# Z-Dax Crypto Platfrom Development Setup

## System requirement

* Docker, docker-compose
* Postgresql or Yugabyte (RDS)
* Redis (K/V Database)
* QuestDB (Time-Series Database)
* Kafka or Redpanda

## Generate JWT Key

First create secrets folder if it's not exists

```bash
mkdir config/secrets
```

Then generate jwt key
Note: Don't use passphrase

```bash
ssh-keygen -t rsa -b 4096 -m PEM -f config/secrets/rsa-key
openssl rsa -in config/secrets/rsa-key -pubout -outform PEM -out config/secrets/rsa-key.pub
```

## Setup development environment

First you need create environment file

### .env

```bash
PORT=3000

LOG_LEVEL=DEBUG
ENVIRONMENT=production
FINEX_ENV=production

DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASS=example
DATABASE_NAME=peatio_production

INFLUXDB_URL=http://localhost:8086
INFLUXDB_DATABASE=peatio_production

KAFKA_URL=localhost:9092

REDIS_HOST=localhost
REDIS_PORT=6379

ENGINE_PORT=9000
MATCHING_ENGINE_URL=localhost:9000

JWT_PUBLIC_KEY=
```

```bash
export $(cat .env | xargs)
```

## Start development backend requirement

To access into Z-Dax platform you must have an authz and a gateway:

```bash
docker-compose -f config/backend.yml up -d
```

Now all backend for developemnt is ready
Good luck!