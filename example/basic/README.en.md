# KFlow Basic Example

This example demonstrates the basic usage of the KFlow framework, including configuration definition and component implementation, with real file read/write operations.

## Structure

- `workflow.json`: Workflow configuration defining a three-layer execution pipeline
- `components.go`: Component implementations including file reading, config reading, transformation, validation, writing, and logging
- `main.go`: Main program showing how to initialize and execute the workflow
- `data.txt`: Sample input file read by FileReaderComponent
- `output.txt`: Sample output file written by FileWriterComponent

## Workflow Overview

This example workflow contains three layers:

1. Data Preparation (data_preparation)
   - Parallel execution mode
   - Includes file reading and config reading components

2. Data Processing (data_processing)
   - Serial execution mode
   - Includes data transformation and data validation components
   - Depends on the data preparation layer

3. Data Output (data_output)
   - Serial execution mode
   - Includes file writing and log notification components
   - Depends on the data processing layer

## Run the Example

From the project root:

```bash
go run ./example/basic
```

Or from the example directory:

```bash
cd example/basic
go run .
```

> Components support resolving relative paths when running from either the project root or the example directory.

## File I/O Notes

- FileReaderComponent: uses `os.ReadFile` to read `data.txt` and writes the result to shared data under `file_data`
- FileWriterComponent: writes `transformed_data` to `output.txt`; uses `append` to control overwrite vs append

## Notes

- Validation rules (validator) default to `not_empty` and `max_length:500`. Adjust rules or simplify `data.txt` for stricter validation.
- All `Execute` method signatures are `Execute(ctx context.Context, data DataContext)`.

## Core Concepts

- Component: the smallest executable unit in the workflow
- Layer: logical grouping of components with execution mode and dependencies
- Mode: supports serial, parallel, and async execution
- Critical Component: components marked critical failing will cause the entire workflow to fail