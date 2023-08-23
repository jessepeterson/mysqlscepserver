FROM gcr.io/distroless/static

ARG TARGETOS TARGETARCH

COPY mysqlscepserver-$TARGETOS-$TARGETARCH /app/mysqlscepserver

EXPOSE 8080

WORKDIR /app

ENTRYPOINT ["/app/mysqlscepserver"]
