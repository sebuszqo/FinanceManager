# FinanceManager

# Backend Setup Instructions

## Prerequisites

Ensure you have the following tools installed on your system:

- **Docker**: [Install Docker](https://docs.docker.com/get-docker/)
- **Docker Compose**: [Install Docker Compose](https://docs.docker.com/compose/install/)

## Setup Steps

### 1. Clone the repository:
```bash
git clone https://github.com/sebuszqo/FinanceManager
cd FinanceManager
```

## Setup Steps

You will need to create two files: db_user.txt and db_password.txt. These files will store the database username and password, which will be used by Docker to create the database.

### 3. Prepare the .env.docker, db_user.txt, db_password.txt files 
The .env.docker file should include all the environment variables required by your Go application, such as database credentials, host configuration, etc. Here’s an example of a basic .env file:
```bash
DB_CONNECTION_STRING=host=localhost user=<your_user> password=<password> dbname=<dbname> sslmode=disable
JWT_SECRET=<JWT_SECRET>
EMAIL_PASSWORD=<email_passoword>
TEMPLATES_DIR=<path to: FinanceManager/internal/email/templates>
EMAIL_ADDRESS=<email_address_configured_with_google_SMTP>
```
Make sure the .env.docker file is in the same directory as your docker-compose.yml and Dockerfile.

Prepare `db_user.txt`, by adding db user to file:
```bash
  db_user
```


Prepare `db_password.txt`, by adding db password to file:
```bash
  db_password
```

### 4. Build and run the Docker containers
Now, run the following command to build and start the containers defined in the docker-compose.yml file:
```bash
docker-compose up --build
```

This will:

- Build the Go application using the Dockerfile.
- Start two services:
  - db: A PostgreSQL container using the credentials specified in the db_user.txt and db_password.txt files.
  - app: The Go application, which will connect to the database service.

### 5. Check logs
You can view the application logs to ensure everything is running smoothly by using:

```bash
docker-compose logs -f
```

### 6. Access the application
Once the containers are running, the application should be accessible on localhost:8080:

- API URL: http://localhost:8080/ready
- You can use a tool like Postman, curl, or your browser to make requests to the API.

### 7. When you're done or need to stop the containers, you can bring them down with:
```bash
docker-compose down
```

### 8. Persistent storage
- The database data is stored in a Docker volume named db_data, which is mounted inside the PostgreSQL container at /var/lib/postgresql/data. This means that even if you stop or remove the container, your data will persist in this volume.

- The volume is defined in the docker-compose.yml as follows:
```bash 
volumes:
  db_data:
```
Note: This data is stored inside the Docker environment, not directly on your local file system. If you need to remove the data, you must remove the db_data volume manually using:
```bash
docker volume rm <your_project>_db_data
```