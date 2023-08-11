FROM golang

COPY . /app/
WORKDIR /app

RUN CGO_ENABLED=0 go build -a -ldflags '-w -s'

RUN chmod +x ./github_release_notes

CMD ["/app/github_release_notes"]