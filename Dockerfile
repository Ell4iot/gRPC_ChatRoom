FROM ubuntu:latest

# Install basic packages
RUN apt update
RUN apt-get update && apt-get install -y apt-transport-https
RUN apt install -y openssl ca-certificates vim make gcc golang-go protobuf-compiler netcat iputils-ping iproute2

# Set up certificates
ARG cert_location=/usr/local/share/ca-certificates
RUN mkdir -p ${cert_location}
# Get certificate from "github.com"
RUN openssl s_client -showcerts -connect github.com:443 </dev/null 2>/dev/null|openssl x509 -outform PEM > ${cert_location}/github.crt
# Get certificate from "proxy.golang.org"
RUN openssl s_client -showcerts -connect google.golang.org:443 </dev/null 2>/dev/null|openssl x509 -outform PEM >  ${cert_location}/google.golang.crt
# Update certificates
RUN update-ca-certificates

# Install go extensions for protoc
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
ENV PATH="$PATH:/root/go/bin"
ENV PATH="$PATH:$(go env GOPATH)/bin"
ENV PATH=$PATH:$GOROOT/bin

COPY chat_system /usr/lib/go-1.18/src/p1
WORKDIR /usr/lib/go-1.18/src/p1
RUN go mod init chat
RUN go mod tidy
WORKDIR /usr/lib/go-1.18/src/p1/chat
RUN protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative *.proto
WORKDIR /usr/lib/go-1.18/src/p1
RUN go build -o chat_server server.go
RUN go build -o chat_client client.go
