# Warden Quick Start Examples

> 🌐 **Language / 语言**: [English](README.md) | [中文](README.zhCN.md)

This directory contains two Warden usage examples of different complexity levels to help you get started quickly.

## 🐳 Docker Image

All examples use pre-built images from [GitHub Container Registry (GHCR)](https://github.com/soulteary/warden/pkgs/container/warden):

- **Default Image**: `ghcr.io/soulteary/warden:latest`
- **Customization**: You can override the image by setting `WARDEN_IMAGE` and `WARDEN_IMAGE_TAG` environment variables

For production deployments, it's recommended to use a specific version tag instead of `latest`.

## 📚 Example List

### 1. [Simple Example](./basic/) - Basic Usage

**Suitable Scenarios**:
- Quick testing and development
- Using only local data files
- Learning basic functionality

**Includes**:
- ✅ Local data file configuration
- ✅ Basic Docker Compose deployment
- ✅ Simple startup script
- ✅ Complete usage documentation

**Quick Start**:
```bash
cd basic
# Pull the latest image (optional)
docker-compose pull
docker-compose up -d
```

[View Detailed Documentation →](./basic/README.md)

### 2. [Advanced Example](./advanced/) - Full Features

**Suitable Scenarios**:
- Isolated development deployment with a mock remote source; production needs separate HTTPS, authentication and network configuration
- Need remote API data source
- Complete monitoring and testing

**Includes**:
- ✅ Local + remote data sources
- ✅ Redis cache and distributed locks
- ✅ Scheduled tasks for automatic synchronization
- ✅ Mock remote API service
- ✅ Complete Docker Compose configuration
- ✅ Automated test scripts
- ✅ Multiple data merging strategy demonstrations

**Quick Start**:
```bash
cd advanced
cp env.example .env
# Pull the latest image (optional)
docker-compose pull
docker-compose up -d
```

[View Detailed Documentation →](./advanced/README.md)

## 🎯 Selection Guide

### Choose Simple Example if you:
- Are using Warden for the first time
- Only need local data files
- Want to quickly verify functionality
- Are testing in a development environment

### Choose Advanced Example if you:
- Need to fetch data from remote APIs
- Need to understand complete data merging strategies
- Want to learn remote-source configuration in an isolated development environment
- Need a complete monitoring and testing solution

## 🚀 Quick Comparison

| Feature | Simple Example | Advanced Example |
|---------|---------------|------------------|
| Local Data File | ✅ | ✅ |
| Remote API | ❌ | ✅ |
| Redis Cache | ✅ | ✅ |
| Scheduled Tasks | ✅ | ✅ |
| Mock API | ❌ | ✅ |
| Test Scripts | ❌ | ✅ |
| Complete Configuration | ❌ | ✅ |
| Documentation Detail | Basic | Complete |

## 📖 Learning Path

### Beginner Path
1. Start with [Simple Example](./basic/)
2. Understand basic concepts and configuration
3. Test basic functionality
4. Then check [Advanced Example](./advanced/) to learn advanced features

### Experienced User Path
1. Directly check [Advanced Example](./advanced/)
2. Adjust configuration according to needs
3. Refer to main project [README](../README.md) to learn all features

## 🔗 Related Resources

- [Warden Main Documentation](../README.md) - Complete project documentation
- [API Documentation](../openapi.yaml) - OpenAPI specification
- [Configuration Example](../config.example.yaml) - Configuration file reference
- [Code Style Guide](../docs/enUS/CODE_STYLE.md) - Development standards

## 💡 Tips

- All examples can run independently
- Recommend running the simple example first to ensure environment configuration is correct
- Advanced example uses a development-only HTTP mock; review production security requirements separately
- You can modify configuration and data files according to actual needs

## ❓ Need Help?

If you encounter problems:
1. Check the corresponding example's README documentation
2. Check the troubleshooting section in [Main Project README](../README.md)
3. Submit an Issue to the project repository

---

**Enjoy using Warden!** 🎉

