# first step
FROM golang:1.20-alpine AS build_stage

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN go build -o db_forum cmd/main.go

# second step
FROM ubuntu:20.04 AS final_stage

RUN apt-get -y update &&\
    apt-get install -y tzdata &&\
    apt-get install -y postgresql-12 &&\
    rm -rf /var/lib/apt/lists/*

COPY --from=build_stage /build/db/db.sql /app/db.sql
COPY --from=build_stage /build/db/postgresql.conf /app/postgresql.conf

USER postgres

RUN service postgresql start && \
    psql --command "CREATE USER db_forum WITH SUPERUSER PASSWORD 'db_forum';" && \
    createdb -O db_forum db_forum && \
    psql db_forum --echo-all --file /app/db.sql && \
    service postgresql stop

ENV PGVER 12

RUN echo "host all  all    all  trust" >> /etc/postgresql/$PGVER/main/pg_hba.conf && \
    echo "local all  all  trust" >> /etc/postgresql/$PGVER/main/pg_hba.conf && \
    cat /app/postgresql.conf >> /etc/postgresql/$PGVER/main/postgresql.conf

USER root

# VOLUME  ["/etc/postgresql", "/var/log/postgresql", "/var/lib/postgresql"]

EXPOSE 5432
EXPOSE 5000

COPY --from=build_stage /build/db_forum /app/

WORKDIR /app

CMD service postgresql start && ./db_forum
