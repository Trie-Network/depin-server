# Use the official Golang image
FROM golang:1.22

# Set the working directory inside the container
WORKDIR /app

# Copy the entire project into the container
COPY . .

# Download Go module dependencies
RUN go mod download

# Build the Go binary
RUN go build -o depin-server main.go

# Set environment variables (if your API or Rubix node depends on these)
ENV SERVER_PORT=8081
ENV RUBIX_NODE_URL=http://localhost:10500

# Use ENTRYPOINT instead of CMD so the container always runs your server
ENTRYPOINT ["./depin-server"]

