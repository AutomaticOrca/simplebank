# Simple Bank - Backend API

*Securely Atomic, Swiftly Asynchronous.*

## 📖 Project Overview


Simple Bank is a RESTful API backend service built with Go (Golang) and the Gin web framework. It simulates essential banking operations, including user management, account creation, and secure fund transfers. This project was developed to demonstrate practical backend engineering skills, from design and implementation through to successful deployment on the Render.com cloud platform.
This project serves as a comprehensive showcase of backend development skills using Go, covering API design, database interaction, user authentication, asynchronous task handling, unit testing, and cloud deployment.

## ✨ Key Technical Highlights

This project leverages modern technologies to build a robust and efficient backend system:

* **Go & Gin for Performance:** Delivers a high-performance, concurrent API, utilizing native Go goroutines for responsive, non-blocking asynchronous tasks (e.g., email dispatch).
* **PostgreSQL & ACID Transactions:** Ensures financial data integrity and reliability using PostgreSQL, with critical operations like transfers protected by atomic (ACID) database transactions.
* **`sqlc` for Type-Safe Database Code:** Enhances database interaction security and maintainability by generating type-safe Go code directly from SQL queries.
* **PASETO for Secure Authentication:** Implements modern and secure stateless API authentication using PASETO tokens, a simpler and safer alternative to JWT.
* **Docker & `golang-migrate` for Consistency:** Ensures reproducible development and deployment environments with Docker, coupled with version-controlled database schema evolution via `golang-migrate`.
* **Pragmatic Architectural Choices:** Demonstrates practical engineering decisions, notably opting for Go goroutines for asynchronous email processing (over a more complex Asynq/Redis setup). This simplified the architecture, optimized for cost-effectiveness (avoiding Redis free-tier limits), and maintained API responsiveness—reflecting real-world trade-offs.

## ✨ Key Features

* **User Management:**
    * New user registration (passwords hashed with bcrypt).
    * User login with credential validation, issuing PASETO Access and Refresh Tokens.
    * Access Token renewal using Refresh Tokens.
    * Asynchronous email verification for new users (via in-app Goroutines and Gmail SMTP).
    * Email verification status updates.
* **Account Management (Authenticated):**
    * Bank account creation (supports multiple currencies).
    * Query for single account details.
    * List all accounts for a user (with pagination).
* **Transfer Module (Authenticated):**
    * Inter-account fund transfers.
    * Atomic operations for transfers via database transactions (`TransferTx`), ensuring consistency (creates transfer record, updates balances, generates account entries).
    * Query user's transfer history (includes currency information, with pagination).

## 🛠️ Tech Stack

* **Language:** Go (Golang)
* **Web Framework:** Gin Web Framework
* **Database:** PostgreSQL (hosted on Supabase)
* **Database Interaction:** `sqlc` (generates type-safe Go code from SQL)
* **Database Migrations:** `golang-migrate`
* **Authentication:** PASETO (Platform-Agnostic Security Tokens)
* **Configuration Management:** Viper (for local `.env` files) and Environment Variables (`os.Getenv`)
* **Email Sending:** `net/smtp`, `github.com/jordan-wright/email`
* **Asynchronous Tasks:** Go Goroutines (for email sending)
* **Testing:** Go `testing` package, `gomock`/`mockgen` (for unit testing)
* **Containerization:** Docker (for local development and deployment)
* **Deployment:** Render.com
* **Version Control:** Git, GitHub

## 🚀 API Endpoints

| Method | Path                       | Description                      | Auth Required |
| :----- | :------------------------- | :------------------------------- | :------------ |
| POST   | `/users`                   | Create a new user                | No            |
| POST   | `/users/login`             | Log in a user                    | No            |
| GET    | `/users/verify_email`      | Verify user's email              | No            |
| POST   | `/tokens/renew_access`     | Renew Access Token               | Yes           |
| POST   | `/accounts`                | Create a bank account            | Yes           |
| GET    | `/accounts/:id`            | Get single account details       | Yes           |
| GET    | `/accounts`                | List user's accounts (paginated) | Yes           |
| POST   | `/transfers`               | Perform a fund transfer          | Yes           |
| GET    | `/transfers`               | List user's transfers (paginated)| Yes           |

## 🏁 Getting Started

### Prerequisites

* Go (Version `[Your Go Version]` or higher)
* PostgreSQL Database
* `golang-migrate` CLI (Installation: [golang-migrate/migrate/cmd/migrate](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate))
* Docker (Optional, if you prefer to run PostgreSQL or other dependencies via Docker)

### Installation & Running Locally

1.  **Clone the repository:**
    ```bash
    git clone [https://github.com/AutomaticOrca/simplebank.git](https://github.com/AutomaticOrca/simplebank.git)
    cd simplebank
    ```

2.  **Configure Environment Variables:**
    Copy the `.env.example` file to `.env` and fill in your configuration details:
    ```bash
    cp .env.example .env
    ```
    You'll need to configure variables such as:
    * `DB_DRIVER` (e.g., `postgres`)
    * `DB_SOURCE` (e.g., `postgresql://user:password@localhost:5432/simple_bank?sslmode=disable`)
    * `TOKEN_SYMMETRIC_KEY` (a random string of at least 32 characters)
    * `ACCESS_TOKEN_DURATION` (e.g., `15m`)
    * `REFRESH_TOKEN_DURATION` (e.g., `24h`)
    * `EMAIL_SENDER_NAME` (e.g., `Simple Bank Support`)
    * `EMAIL_SENDER_ADDRESS` (your Gmail address)
    * `EMAIL_SENDER_PASSWORD` (your Gmail App Password)
    * `HTTP_SERVER_ADDRESS` (e.g., `0.0.0.0:8080`)
    * `CLIENT_ORIGIN` (Frontend URL for email verification links, e.g., `http://localhost:3000`)

3.  **Run Database Migrations:**
    Ensure your PostgreSQL service is running and the database is created. Then, execute:
    ```bash
    make migrateup
    ```
    To roll back migrations:
    ```bash
    make migratedown
    ```

4.  **Install Dependencies:**
    ```bash
    go mod tidy
    ```

5.  **Run the Server:**
    ```bash
    make server
    # Or directly:
    # go run main.go
    ```
    The server will start on the address specified in `HTTP_SERVER_ADDRESS` (e.g., `http://localhost:8080`).

### Running Tests

To run all unit tests:
```bash
make test
```

# API Documentation

## Basic Information
- Base URL: `http://localhost:8080`
- All requests and responses use JSON format
- Authentication: Bearer Token

## Authentication

### 1. User Registration
- **Endpoint**: `POST /users`
- **Description**: Create a new user account
- **Request Body**:
```json
{
    "username": "string",     // Required, alphanumeric
    "password": "string",     // Required, minimum 6 characters
    "full_name": "string",    // Required
    "email": "string"         // Required, valid email format
}
```
- **Response**: 201 Created
```json
{
    "username": "string",
    "full_name": "string",
    "email": "string",
    "password_changed_at": "timestamp",
    "created_at": "timestamp"
}
```

### 2. User Login
- **Endpoint**: `POST /users/login`
- **Description**: User login and obtain access token
- **Request Body**:
```json
{
    "username": "string",     // Required
    "password": "string"      // Required
}
```
- **Response**: 200 OK
```json
{
    "session_id": "uuid",
    "access_token": "string",
    "access_token_expires_at": "timestamp",
    "refresh_token": "string",
    "refresh_token_expires_at": "timestamp",
    "user": {
        "username": "string",
        "full_name": "string",
        "email": "string",
        "password_changed_at": "timestamp",
        "created_at": "timestamp"
    }
}
```

### 3. Email Verification
- **Endpoint**: `GET /users/verify_email`
- **Description**: Verify user email
- **Query Parameters**:
    - `email_id`: Verification email ID
    - `secret_code`: Verification code

### 4. Refresh Access Token
- **Endpoint**: `POST /tokens/renew_access`
- **Description**: Obtain new access token using refresh token

## Account Management (Authentication Required)

### 1. Create Account
- **Endpoint**: `POST /accounts`
- **Description**: Create a new bank account
- **Headers**: `Authorization: Bearer <access_token>`
- **Request Body**:
```json
{
    "currency": "string"  // Required, supported currency type
}
```

### 2. Get Account
- **Endpoint**: `GET /accounts/:id`
- **Description**: Get account information by ID
- **Headers**: `Authorization: Bearer <access_token>`
- **Path Parameters**: `id` - Account ID

### 3. List Accounts
- **Endpoint**: `GET /accounts`
- **Description**: Get all accounts for the current user
- **Headers**: `Authorization: Bearer <access_token>`

## Transfer Operations (Authentication Required)

### 1. Create Transfer
- **Endpoint**: `POST /transfers`
- **Description**: Create a new transfer transaction
- **Headers**: `Authorization: Bearer <access_token>`
- **Request Body**:
```json
{
    "from_account_id": "integer",
    "to_account_id": "integer",
    "amount": "decimal",
    "currency": "string"
}
```

### 2. List Transfers
- **Endpoint**: `GET /transfers`
- **Description**: Get transfer history for the current user
- **Headers**: `Authorization: Bearer <access_token>`

## Error Responses
All APIs return the following format when an error occurs:
```json
{
    "error": "Error message description"
}
```

## Status Codes
- 200: Success
- 201: Created
- 400: Bad Request
- 401: Unauthorized
- 403: Forbidden
- 404: Not Found
- 409: Conflict
- 500: Internal Server Error

## Security Features
1. All passwords are encrypted before storage
2. JWT token-based authentication
3. Token refresh mechanism
4. Email verification process
5. Authentication required for all sensitive operations

## CORS Configuration
API supports cross-origin requests from:
- `http://localhost:3000`
- `https://simplebank-frontend.vercel.app`

This API documentation covers the main endpoints and functionality. If you need more detailed information about any specific endpoint, I can provide additional details.
