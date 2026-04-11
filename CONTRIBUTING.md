# Contributing to Bore

Thanks for your interest in contributing to Bore! This guide covers how to submit changes.

## Getting Started

1. **Fork** the repository on GitHub
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/<your-username>/bore.git
   cd bore
   ```
3. **Add the upstream remote** so you can keep your fork in sync:
   ```bash
   git remote add upstream https://github.com/hyperplex-tech/bore.git
   ```
4. **Set up dependencies:**
   ```bash
   ./scripts/dev-setup.sh
   ```

See [DEVELOPERS.md](DEVELOPERS.md) for full build prerequisites and Makefile targets.

## Making Changes

1. **Sync your fork** with upstream before starting work:
   ```bash
   git fetch upstream
   git checkout main
   git merge upstream/main
   ```
2. **Create a feature branch** from `main`:
   ```bash
   git checkout -b my-feature
   ```
3. **Make your changes.** Keep commits focused — one logical change per commit.
4. **Test your changes:**
   ```bash
   make test
   make lint
   ```
5. **Push** your branch to your fork:
   ```bash
   git push origin my-feature
   ```
6. **Open a Pull Request** against `hyperplex-tech/bore:main` on GitHub.

## Pull Request Guidelines

- Keep PRs focused on a single change. If you're fixing a bug and also want to refactor something nearby, open separate PRs.
- Write a clear description of what your PR does and why.
- Make sure CI passes (tests, lint, build) before requesting review.
- If your PR adds a new feature, include tests for it.
- If your PR changes user-facing behavior, update the relevant docs (README, DEVELOPERS.md).

## Development Workflow

Start the dev daemon (uses a separate socket so it won't conflict with a production install):

```bash
make dev
```

In another terminal, interact with it:

```bash
BORE_SOCKET=$(pwd)/bin/bored-dev.sock ./bin/bore status
```

Or launch the desktop app against the dev daemon:

```bash
make dev-desktop
```

## Reporting Issues

Open an issue on [GitHub Issues](https://github.com/hyperplex-tech/bore/issues). Include:

- What you expected to happen
- What actually happened
- Steps to reproduce
- Platform and version (`bore daemon status`)

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
