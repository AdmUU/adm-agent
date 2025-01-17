# ADM Agent

[![Go Version](https://img.shields.io/github/go-mod/go-version/admuu/adm-agent)](https://github.com/admuu/adm-agent)
[![License](https://img.shields.io/github/license/admuu/adm-agent)](https://github.com/admuu/adm-agent/blob/main/LICENSE)

This is the agent of [Admin.IM](https://www.admin.im) platform. It integrates seamlessly with ADM's API service, providing real-time online ping/tcping latency testing, website speed testing, route tracking, etc. It also includes configuration management, service control, and secure socket connection functions.
## Features

- 🌐 Integrate multiple network detection functions
- 🛡️ Runs safely with minimal system privileges
- 🚀 Easy service management (start, stop, status)
- 🔒 Secure socket.io connections
- 🔄 Automatic updates checking
- 📤 Optional sharing capabilities
- 🎯 Multi-platform support (Cross-platform compatibility)


## Installation

### Online deployment

#### Prerequisites

- Linux systems that support systemd
- Install using sudo privileges

```bash
bash <(curl -fsSL https://get.admin.im) -a https://your_domain -k your_key -s your_secret -share yes|no
```
You can get the key and secret after deploying the ADM server.

### Building from Source

#### Prerequisites

- Go 1.21 or higher

```bash
git clone https://github.com/admuu/adm-agent.git
cd adm-agent
go build
```

## Usage

### Basic Commands

```bash
# Register a node
./adm-agent register --config /path/to/config.yaml -a https://your_domain -k your_key -s your_secret

# Run
./adm-agent --config /path/to/config.yaml

# Install as a system service
./adm-agent install

# Uninstall system service
./adm-agent uninstall

# Check agent service status
./adm-agent status

# Stop the agent service
./adm-agent stop

# Check version
./adm-agent --version
```

### Configuration

Create a configuration file (e.g., `config.yaml`) with the following structure:

```yaml
app:
  env: "prod"  # or "dev" for development environment

api:
  url: "your-api-url"
  authcode: "your-auth-code"
  did: "your-node-id"

share:
  enable: "yes"  # or "no" to disable sharing
  authcode: "your-share-auth-code"
  did: "your-share-node-id"

ip:
    prefer: ""  # ip priority, "ipv4" or "ipv6"
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

Copyright © 2024 - 2025 Admin.IM <dev@admin.im>

This project is under the GNU General Public License Version 3. See the [LICENSE](https://github.com/admuu/adm-agent/blob/main/LICENSE) file for the full license text.

## Support

For support and more information, please visit [https://www.admin.im](https://www.admin.im) or create an issue in the GitHub repository.