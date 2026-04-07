package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	serviceName := flag.String("name", "", "Name of the service (e.g., user, payment)")
	flag.Parse()

	if *serviceName == "" {
		fmt.Println("Please provide a service name using -name flag")
		os.Exit(1)
	}

	// Create service directory structure
	basePath := filepath.Join("services", *serviceName+"-service")
	dirs := []string{
		"cmd",
		"internal/core/domain",
		"internal/core/ports/inbound",
		"internal/core/ports/outbound",
		"internal/service",
		"internal/adapters/primary",
		"internal/adapters/secondary",
		"pkg/types",
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(basePath, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			fmt.Printf("Error creating directory %s: %v\n", dir, err)
			os.Exit(1)
		}
	}

	// Create an empty README.md
	readmePath := filepath.Join(basePath, "README.md")
	readmeContent := fmt.Sprintf(`# %s service

This service handles all %s-related operations in the system.

## Architecture

The service follows Clean Architecture principles with the following structure:

` + "```" + `
services/%s-service/
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
` + "```" + `

### Layer Responsibilities

1. **Core Layer** (` + "`internal/core/`" + `)
   - Domain models and port interfaces (inbound/outbound)
   - Pure business logic contracts, no infrastructure details

2. **Service Layer** (` + "`internal/service/`" + `)
   - Implements business logic
   - Uses outbound port interfaces
   - Coordinates between different parts of the system

3. **Adapters Layer** (` + "`internal/adapters/`" + `)
   - ` + "`primary/`" + `: Inbound adapters (HTTP/gRPC, CLI, etc.)
   - ` + "`secondary/`" + `: Outbound adapters (DB, MQ, external APIs)

4. **Public Types** (` + "`pkg/types/`" + `)
   - Contains shared types and models
   - Can be imported by other services

## Key Benefits

1. **Dependency Inversion**: Services depend on interfaces, not implementations
2. **Separation of Concerns**: Each layer has a specific responsibility
3. **Testability**: Easy to mock dependencies for testing
4. **Maintainability**: Clear boundaries between components
5. **Flexibility**: Easy to swap implementations without affecting business logic
`, *serviceName, *serviceName, *serviceName)

	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		fmt.Printf("Error creating README.md: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully created %s service structure in %s\n", *serviceName, basePath)
	fmt.Println("\nDirectory structure created:")
	fmt.Printf(`
services/%s-service/
├── cmd/                    # Application entry points
├── internal/              # Private application code
│   ├── core/             # Domain + ports (hexagon center)
│   │   ├── domain/
│   │   └── ports/
│   │       ├── inbound/
│   │       └── outbound/
│   ├── service/          # Usecase implementations
│   └── adapters/         # Primary/secondary adapters
│       ├── primary/
│       └── secondary/
├── pkg/                  # Public packages
│   └── types/           # Shared types and models
└── README.md            # This file
`, *serviceName, *serviceName)
} 