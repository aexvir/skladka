FROM golang:1.23 AS dependencies

WORKDIR /skladka

COPY go.mod go.sum ./
RUN go mod download \
    && go install github.com/magefile/mage@latest

FROM dependencies AS builder

ARG BUILD_BRANCH
ARG BUILD_REV

WORKDIR /skladka
COPY . .

RUN GOFLAGS="-buildvcs=false" \
    CI=true \
    BUILD_BRANCH=${BUILD_BRANCH} \
    BUILD_REV=${BUILD_REV} \
    mage build

FROM gcr.io/distroless/static

COPY --from=builder /skladka/bin/skladka /bin/skladka

CMD ["skladka"]
EXPOSE 3000
