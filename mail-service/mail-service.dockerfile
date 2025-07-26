FROM alpine:latest

WORKDIR /app

COPY mailApp .
COPY templates ./templates

CMD ["./mailApp"] 
