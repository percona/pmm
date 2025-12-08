# PMM Development Guide for AI Agents

## Purpose

This PR extracts AI agent configuration files to provide structured guidelines for AI coding agents working with the PMM codebase. These files help agents understand:
- The overall repository structure and component organization
- Component-specific development patterns and conventions
- Best practices for contributing to PMM

The goal is to improve AI agent effectiveness when making code changes, reviewing code, or providing guidance on PMM development.

## Project Overview

Percona Monitoring and Management (PMM) is an open-source database monitoring solution with a client-server architecture. This is a **monorepository** containing multiple PMM Components, APIs, documentation, and build scripts.

Every component is written in Go, with the exception of the UI, which is based on TypeScript. Each component has its own directory at the root of the repository.

### Core Components

- **pmm-managed** (`/managed`) - Backend service managing PMM Server configuration, exposes gRPC/REST APIs
- **pmm-agent** (`/agent`) - Client-side agent that runs exporters and collects metrics via VMAgent
- **pmm-admin** (`/admin`) - CLI tool for managing monitored services, wraps pmm-agent functionality
- **qan-api2** (`/qan-api2`) - Query Analytics API service
- **APIs** (`/api`) - Protobuf definitions and generated clients for all services
- **API Tests** (`/api-tests`) - Integration tests for PMM APIs
- **UI** (`/ui`) - React-based PMM frontend
- **VMProxy** (`/vmproxy`) - VMProxy is a stateless reverse proxy for VictoriaMetrics
- **Utils** (`/utils`) - Shared utility libraries for PMM components
- **API Documentation** (`/docs`) - PMM API documentation
- **Documentation** (`/documentation`) - Documentation source files

### Other Directories
- **build** (`/build`) - Build scripts and Dockerfiles
- **scripts** (`/scripts`) - Utility scripts for development and maintenance

# AI Agent Instructions

The following guidelines are intended to help AI coding agents contribute effectively to the PMM codebase. 

## Component-Specific Guidelines

Each PMM component can have its own `AGENT.md` file in its respective directory, providing detailed development guidelines specific to that component. This file serves as a general overview and points to these component-specific instructions.

Currently available component guidelines:
- [managed/AGENT.md](../managed/AGENT.md) - Comprehensive AI-driven `pmm-managed` development guidelines

Additional components may add their own `AGENT.md` files as needed (e.g., `agent/AGENT.md`, `ui/AGENT.md`, etc.).
