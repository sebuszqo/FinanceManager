# Start from the official Go image
FROM golang:1.22

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the Working Directory inside the container
COPY . .

# Copy the .env file into the container
COPY .env.docker .env

# Build the Go app
RUN go build -o main ./cmd/FinanceManager

# Copy email templates to a specific directory inside the container
COPY internal/email/templates /app/internal/email/templates

# Expose port 8080 to the outside world
EXPOSE 8080

# Run the executable
CMD ["./main"]
