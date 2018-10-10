FROM golang:1.9-alpine
RUN mkdir -p /kubetastic/bin
COPY dist/bin /kubetastic/bin