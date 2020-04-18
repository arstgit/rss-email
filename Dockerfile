FROM golang
COPY . /app
WORKDIR /app
RUN go build
ENTRYPOINT [ "/app/rss-email" ]