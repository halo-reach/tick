# Contributing to Tick

Thank you for your interest in contributing to Tick!

## How to Contribute

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/halo-reach/tick.git
   cd tick
   ```
3. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```
4. **Make your changes** and commit with clear messages
5. **Push to your fork**:
   ```bash
   git push origin feature/your-feature-name
   ```
6. **Open a Pull Request** on GitHub

## Development Setup

```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Build
make build
```

## Code Style

- Follow Go coding conventions (run `go fmt` before committing)
- Write tests for new features
- Keep commits atomic and well-described

## Reporting Issues

- Use GitHub Issues to report bugs or request features
- Include code samples and expected/actual behavior when reporting bugs
- Check existing issues before creating a new one

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
