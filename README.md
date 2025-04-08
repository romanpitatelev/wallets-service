# Wallet Management Service (Golang)

A high-performance REST API for managing digital wallets with multi-currency support, JWT authentication, and Kafka event streaming.

Go
PostgreSQL
Kafka
Features

✨ Core Operations
- Create/read/update/delete wallets
- Deposit/withdraw funds
- Multi-currency transfers with auto-conversion

🛡️ Security
- JWT authentication
- Transaction-level integrity checks

📊 Observability
- Prometheus metrics endpoint
- Request duration tracking
- Transaction success/failure monitoring

## API Reference
- /wallets	    POST	  Create new wallet
- /wallets/{id}	GET	Get wallet details
- /wallets/{id}	DELETE	Delete wallet
- /deposit	PUT	Add funds to wallet
- /withdraw	PUT	Remove funds from wallet
- /transfer	PUT	Transfer between wallets
- /metrics	GET	Prometheus metrics endpoint

Tech Stack Backend
• Chi router 
• PGX driver 
• Sarama Kafka client

Infrastructure
• PostgreSQL 15+ 
• Apache Kafka 3.x 
• Prometheus

Tooling
• Docker multi-stage builds 
• migrations 
• golangci-lint
