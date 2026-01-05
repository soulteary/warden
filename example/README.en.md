# Warden Quick Start Examples

> 🌐 **Language / 语言**: [English](README.en.md) | [中文](README.md)

This directory contains two Warden usage examples of different complexity levels to help you get started quickly.

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
docker-compose up -d
```

[View Detailed Documentation →](./basic/README.en.md)

### 2. [Advanced Example](./advanced/) - Full Features

**Suitable Scenarios**:
- Production environment deployment reference
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
cp .env.example .env
docker-compose up -d
```

[View Detailed Documentation →](./advanced/README.en.md)

## 🎯 Selection Guide

### Choose Simple Example if you:
- Are using Warden for the first time
- Only need local data files
- Want to quickly verify functionality
- Are testing in a development environment

### Choose Advanced Example if you:
- Need to fetch data from remote APIs
- Need to understand complete data merging strategies
- Are preparing to deploy to production
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
3. Refer to main project [README](../README.en.md) to learn all features

## 🔗 Related Resources

- [Warden Main Documentation](../README.en.md) - Complete project documentation
- [API Documentation](../openapi.yaml) - OpenAPI specification
- [Configuration Example](../config.example.yaml) - Configuration file reference
- [Code Style Guide](../CODE_STYLE.en.md) - Development standards

## 💡 Tips

- All examples can run independently
- Recommend running the simple example first to ensure environment configuration is correct
- Advanced example includes complete production environment best practices
- You can modify configuration and data files according to actual needs

## ❓ Need Help?

If you encounter problems:
1. Check the corresponding example's README documentation
2. Check the troubleshooting section in [Main Project README](../README.en.md)
3. Submit an Issue to the project repository

---

**Enjoy using Warden!** 🎉

