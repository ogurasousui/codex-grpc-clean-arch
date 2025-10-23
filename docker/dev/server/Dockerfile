FROM golang:1.24-bullseye

WORKDIR /workspace

RUN apt-get update \
    && apt-get install -y --no-install-recommends curl git \
    && rm -rf /var/lib/apt/lists/*

RUN go install github.com/air-verse/air@v1.52.3

# Cache dirs are mounted via docker-compose for quicker rebuilds
VOLUME ["/go/pkg/mod", "/root/.cache/go-build"]

ENV CONFIG_PATH=assets/local-compose.yaml

CMD ["air", "-c", ".air.toml"]
