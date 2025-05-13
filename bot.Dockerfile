# я пытался сделать, чтобы докер файлы лежали в папке cmd
# Не получилось, docker build не может найти go.mod
FROM golang:1.23-alpine AS build 

WORKDIR /app 


COPY go.mod  ./
COPY go.sum ./

RUN go mod download 

COPY . . 

RUN go build -o main cmd/bot/main.go  

FROM alpine:3.21.3	

WORKDIR /app 

COPY --from=build /app/main .

RUN apk add --no-cache bash=5.2.37-r0

CMD ["./main"]
