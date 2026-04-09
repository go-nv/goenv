# Migration Guide: goenv v2 to v3

## Overview

goenv v3 is fully backward compatible with v2. Most users can migrate immediately without any changes to their workflows or scripts.

## Key Changes

### Architecture
- **v2**: Shell script-based implementation
- **v3**: Go-based CLI for improved performance and reliability

### Compatibility
✅ All v2 commands work identically in v3  
✅ `.go-version` files are fully compatible  
✅ Environment variables (`GOENV_ROOT`, `GOPATH`, `GOROOT`) work the same  
✅ Plugin system remains compatible  
✅ Shell initialization (`goenv init`) unchanged  

## Migration Steps

### For Individual Users

1. **If using Homebrew:**
   ```bash
   # Upgrade to v3
   brew upgrade goenv
   
   # Verify installation
   goenv --version
   ```

2. **If using Git installation:**
   ```bash
   cd ~/.goenv
   git fetch --all
   git checkout v3
   
   # Restart your shell
   exec $SHELL
   
   # Verify installation
   goenv --version
   ```

3. **Test your setup:**
   ```bash
   # Check current version
   goenv version
   
   # List installed versions
   goenv versions
   
   # Install a new version (if needed)
   goenv install 1.21.0
   ```

### For CI/CD Systems

#### AWS CodeBuild

Update your `buildspec.yml`:

```yaml
phases:
  pre_build:
    commands:
      - echo "Installing goenv v3..."
      - git clone https://github.com/go-nv/goenv.git ~/.goenv
      - export GOENV_ROOT="$HOME/.goenv"
      - export PATH="$GOENV_ROOT/bin:$PATH"
      - eval "$(goenv init -)"
      - goenv install 1.21.0
      - goenv global 1.21.0
      - go version
```

#### Docker Containers

Update your Dockerfile:

```dockerfile
# Install goenv v3
RUN git clone https://github.com/go-nv/goenv.git /root/.goenv
ENV GOENV_ROOT=/root/.goenv
ENV PATH=$GOENV_ROOT/bin:$PATH

# Initialize goenv
RUN echo 'eval "$(goenv init -)"' >> /root/.bashrc
RUN bash -c 'eval "$(goenv init -)" && goenv install 1.21.0 && goenv global 1.21.0'
```

#### GitHub Actions

```yaml
- name: Setup goenv
  run: |
    git clone https://github.com/go-nv/goenv.git ~/.goenv
    echo "GOENV_ROOT=$HOME/.goenv" >> $GITHUB_ENV
    echo "$HOME/.goenv/bin" >> $GITHUB_PATH
    
- name: Install Go via goenv
  run: |
    eval "$(goenv init -)"
    goenv install 1.21.0
    goenv global 1.21.0
```

## Staying on v2

If you need to remain on v2 for validation purposes:

### Homebrew
```bash
brew install goenv@2 && brew link goenv@2
```

### Git
```bash
cd ~/.goenv
git checkout master  # v2 is maintained on master branch
```

## Rollback to v2

If you encounter issues with v3:

### Homebrew
```bash
brew unlink goenv # keeps install, but allows for switching
brew install goenv@2 && brew link goenv@2
```

### Git
```bash
cd ~/.goenv
git checkout master
exec $SHELL
```

## Support

- **v3 Issues**: [GitHub Issues](https://github.com/go-nv/goenv/issues)
- **v2 Support**: Maintained until April 2028 or End of Support
- **Questions**: [GitHub Discussions](https://github.com/go-nv/goenv/discussions)

## Breaking Changes

There are **no breaking changes** between v2 and v3. All commands, flags, and behaviors remain consistent.

## Performance Improvements

v3 offers significant performance improvements over v2:
- Faster command execution (Go vs shell scripts)
- Reduced startup time
- More efficient version switching
- Improved error handling and messaging

## Recommended Migration Timeline

- **Immediate**: Individual developer machines
- **1-2 weeks**: Development and staging environments  
- **1-2 months**: Production CI/CD pipelines (after validation)
- **2-6 months**: Critical infrastructure with strict change control
