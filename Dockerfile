# Use the official Golang image
FROM golang:1.22

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./

# Download Go module dependencies
RUN go mod download

# Copy the entire project into the container
COPY . .

# Build the Go binary
RUN go build -o depin-server main.go

# Set environment variables (if your API or Rubix node depends on these)
ENV SERVER_PORT=8081
ENV RUBIX_NODE_URL=http://localhost:10500

# Use ENTRYPOINT instead of CMD so the container always runs your server
ENTRYPOINT ["./depin-server"]

