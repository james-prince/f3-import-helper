FROM golang:alpine AS build
RUN apk update && apk add ca-certificates
WORKDIR /app
ADD . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -tags timetzdata -o ff3-import-helper

FROM scratch AS final
COPY --from=build /app/ff3-import-helper /
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENV TZ='Etc/UTC'
LABEL org.opencontainers.image.source=https://github.com/james-prince/ff3-import-helper

HEALTHCHECK --interval=10s --timeout=3s --start-period=20s \
  CMD ["/ff3-import-helper", "healthcheck"]

ENTRYPOINT ["/ff3-import-helper"]