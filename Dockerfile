#syntax=docker/dockerfile:1.5.1

FROM golang:1.21-alpine AS build
WORKDIR /go/src/exporter_exporter
COPY . .
ENV CGO_ENABLED=0
ENV GOOS=linux

RUN go mod download ;\
    go build -trimpath

# Using nonroot variant for slightly better security posture
FROM gcr.io/distroless/static-debian12:nonroot AS runtime
# Copy the binary and default config from the build stage
COPY --from=build /go/src/exporter_exporter/exporter_exporter /exporter_exporter
# Expose default port; override with --port flag if needed
EXPOSE 9999
ENTRYPOINT [ "/exporter_exporter" ]
CMD [ "--config.file=/etc/exporter_exporter/config.yml" ]
