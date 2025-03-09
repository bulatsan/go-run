# go-runner

A lightweight and flexible runner abstraction for Go applications that makes concurrent task execution simple.

## Installation

```bash
go get github.com/bulatsan/go-runner
```

## Features

- Simple interface with a single `Run` method
- Create runners from any function
- Combine multiple runners to execute in parallel
- Automatic cancellation propagation when errors occur
- Easy error handling
- Zero dependencies

## Usage

### Basic Usage

```go
package main

import (
	"context"
	"fmt"
	"time"

	runner "github.com/bulatsan/go-runner"
)

func main() {
	// Create a simple runner
	simpleRunner := runner.New(func(ctx context.Context) error {
		fmt.Println("Running a task...")
		time.Sleep(100 * time.Millisecond)
		fmt.Println("Task completed!")
		return nil
	})

	// Run it
	err := simpleRunner.Run(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
```

### Returning Errors

```go
// Create a runner that always returns an error
errRunner := runner.Err(fmt.Errorf("something went wrong"))

// Or create a runner that always succeeds
okRunner := runner.OK()
```

### Running Multiple Tasks in Parallel

```go
package main

import (
	"context"
	"fmt"
	"time"

	runner "github.com/bulatsan/go-runner"
)

func main() {
	// Create several runners
	runner1 := runner.New(func(ctx context.Context) error {
		fmt.Println("Runner 1 started")
		time.Sleep(100 * time.Millisecond)
		fmt.Println("Runner 1 completed")
		return nil
	})

	runner2 := runner.New(func(ctx context.Context) error {
		fmt.Println("Runner 2 started")
		time.Sleep(200 * time.Millisecond)
		fmt.Println("Runner 2 completed")
		return nil
	})

	// Join them to run in parallel
	combined := runner.Join(runner1, runner2)
	
	// Run them all at once
	err := combined.Run(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("All runners completed successfully")
	}
}
```

### Error Handling and Cancellation

When using `Join`, if any runner returns an error, all other runners will be cancelled:

```go
package main

import (
	"context"
	"fmt"
	"time"
	"errors"

	runner "github.com/bulatsan/go-runner"
)

func main() {
	// Create a runner that fails quickly
	failingRunner := runner.New(func(ctx context.Context) error {
		fmt.Println("Failing runner started")
		time.Sleep(50 * time.Millisecond)
		fmt.Println("Failing runner returning error")
		return errors.New("something went wrong")
	})

	// Create a runner that takes longer
	slowRunner := runner.New(func(ctx context.Context) error {
		fmt.Println("Slow runner started")
		select {
		case <-time.After(1 * time.Second):
			fmt.Println("Slow runner completed")
			return nil
		case <-ctx.Done():
			fmt.Println("Slow runner was cancelled:", ctx.Err())
			return ctx.Err()
		}
	})

	// Join them
	combined := runner.Join(failingRunner, slowRunner)
	
	// The failing runner will cause the slow runner to be cancelled
	err := combined.Run(context.Background())
	fmt.Printf("Error: %v\n", err) // Will print the error from failingRunner
}
```

## Real-world Example: HTTP Server with Graceful Shutdown

```go
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	runner "github.com/bulatsan/go-runner"
)

func main() {
	// Create an HTTP server
	server := &http.Server{
		Addr:    ":8080",
		Handler: http.DefaultServeMux,
	}

	// Set up routes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, world!")
	})

	// Create a runner for the HTTP server
	serverRunner := runner.New(func(ctx context.Context) error {
		fmt.Println("Starting HTTP server on :8080")
		
		// Start the server
		err := server.ListenAndServe()
		if err != http.ErrServerClosed {
			return err
		}
		return nil
	})

	// Create a runner for graceful shutdown
	shutdownRunner := runner.New(func(ctx context.Context) error {
		// Wait for context cancellation (e.g., SIGINT or SIGTERM)
		<-ctx.Done()
		
		fmt.Println("Shutting down HTTP server...")
		
		// Create a timeout context for shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// Perform graceful shutdown
		return server.Shutdown(shutdownCtx)
	})

	// Create a runner that handles OS signals
	signalCtx, signalCancel := context.WithCancel(context.Background())
	signalRunner := runner.New(func(ctx context.Context) error {
		// Set up signal handling
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
		
		// Wait for SIGINT or SIGTERM
		select {
		case sig := <-signals:
			fmt.Printf("Received signal: %v\n", sig)
			signalCancel() // Cancel the context to trigger shutdown
		case <-ctx.Done():
			// Context was cancelled elsewhere
		}
		
		return nil
	})

	// Join all runners
	combined := runner.Join(serverRunner, shutdownRunner, signalRunner)
	
	// Run everything
	if err := combined.Run(signalCtx); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("Server stopped gracefully")
}
```

## License

See the [LICENSE](LICENSE) file for details.