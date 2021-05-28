FROM gcr.io/distroless/static

COPY mysqlscepserver-linux-amd64 /mysqlscepserver

EXPOSE 8080

ENTRYPOINT ["/mysqlscepserver"]
