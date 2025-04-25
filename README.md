# convertSLX - Simulink File Version Conversion Tool

A simple tool to convert Simulink `.slx` files saved in a newer version back to a previous version by updating internal XML metadata.

## Prerequisites

- Go 1.20+ (for the Go CLI)
- `github.com/beevik/etree` Go module

## Installation

```sh
go mod tidy
go build
```

## Usage

```sh
convertSLX.exe [options] <input.slx|.sldd|.mldatx or directory>
```

### Options:

```
  -d, --directory    Process all .slx/.sldd/.mldatx files in directory recursively
  --r2022a           Set output to R2023b
  --r2022b           Set output to R2024a
  --r2023a           Set output to R2024b
  --r2023b           Set output to R2023a
  --r2024a           Set output to R2022b
  --r2024b           Set output to R2022a
```

### Examples:

```sh
convertSLX.exe --r2023b model.slx                  # Convert a single file to R2023B

convertSLX.exe --r2022a data.sldd                  # Convert a single file to R2022A

convertSLX.exe --2024a -d folder_with_archives    # Convert all .slx, .sldd, or .mldatx files in directory to R2024A
```

## License

MIT © Stuart Alexander