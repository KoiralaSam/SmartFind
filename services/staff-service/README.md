# staff service

This service handles all staff-related operations in the system.

## Architecture

The service follows Clean Architecture principles with the following structure:

```
services/staff-service/
├── cmd/                    # Application entry points
│   └── main.go            # Main application setup
├── internal/              # Private application code
│   ├── core/             # Domain + ports (hexagon center)
│   │   ├── domain/       # Business domain models
│   │   └── ports/        # Inbound/outbound interfaces
│   │       ├── inbound/  # Usecase interfaces
│   │       └── outbound/ # Repository/gateway interfaces
│   ├── service/          # Usecase implementations
│   └── adapters/         # Primary/secondary adapters (hexagon edges)
│       ├── primary/      # Inbound adapters (e.g., HTTP/gRPC handlers)
│       └── secondary/    # Outbound adapters (e.g., Postgres, MQ)
├── pkg/                  # Public packages
│   └── types/           # Shared types and models
└── README.md            # This file
```

### Layer Responsibilities

1. **Core Layer** (`internal/core/`)
   - Domain models and port interfaces (inbound/outbound)
   - Pure business logic contracts, no infrastructure details

2. **Service Layer** (`internal/service/`)
   - Implements business logic
   - Uses outbound port interfaces
   - Coordinates between different parts of the system

3. **Adapters Layer** (`internal/adapters/`)
   - `primary/`: Inbound adapters (HTTP/gRPC, CLI, etc.)
   - `secondary/`: Outbound adapters (DB, MQ, external APIs)

4. **Public Types** (`pkg/types/`)
   - Contains shared types and models
   - Can be imported by other services

## Key Benefits

1. **Dependency Inversion**: Services depend on interfaces, not implementations
2. **Separation of Concerns**: Each layer has a specific responsibility
3. **Testability**: Easy to mock dependencies for testing
4. **Maintainability**: Clear boundaries between components
5. **Flexibility**: Easy to swap implementations without affecting business logic
