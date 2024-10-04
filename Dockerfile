# Start from the official Go image
FROM golang:1.22

# Set the Current Working Directory inside the container
WORKDIR /app

# Install curl to download wait-for-it.sh
RUN apt-get update && apt-get install -y curl

# Download wait-for-it.sh and make it executable
RUN curl -o /wait-for-it.sh https://raw.githubusercontent.com/vishnubob/wait-for-it/master/wait-for-it.sh
RUN chmod +x /wait-for-it.sh

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
CMD ["/wait-for-it.sh", "db:5432", "--", "./main"]

