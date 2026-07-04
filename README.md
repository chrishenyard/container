# Linux Container Implementation in Go and Rust

A lightweight educational project that demonstrates how containerization works by implementing a basic Linux container runtime from scratch in Go and Rust. This project illustrates the core concepts behind Docker and other container technologies, including Linux namespaces, filesystem isolation, and process execution.

## Overview

This project provides a minimal but functional container implementation that:
- Creates isolated process namespaces (PID, UTS, Mount, and optionally User namespaces)
- Uses `chroot` to provide filesystem isolation with a complete Alpine Linux root filesystem
- Supports both root and user mode execution
- Mounts and manages the `/proc` filesystem for process isolation
- Executes arbitrary commands within the isolated environment

## Key Features

- **Namespace Isolation**: Isolates PID, hostname (UTS), and mount namespaces to create process and filesystem boundaries
- **User Namespace Support**: Optional user namespace mapping for running containers with non-root privileges
- **Alpine Linux Root Filesystem**: Includes a complete Alpine Linux `container_rootfs/` for the container runtime environment
- **Dual Mode Operation**: 
  - **Root Mode**: Direct namespace cloning with full isolation
  - **User Mode**: User namespace mapping that allows non-root users to create containers
- **Process Management**: Proper cleanup of mounts and namespace teardown when containers exit

## How It Works

The implementation follows a parent-child execution pattern:

1. **Parent Process**: Requests new namespaces from the Linux kernel using the `clone` syscall with appropriate flags
2. **Child Process**: 
   - Runs within the newly created namespaces
   - Changes hostname for the isolated environment
   - Performs `chroot` into the Alpine filesystem
   - Mounts an isolated `/proc` filesystem
   - Executes the user-specified command
   - Cleans up resources on exit

## Requirements

- Linux operating system (namespaces and chroot are Linux-specific features)
- Go 1.22.2 or later
- Rust 1.85.0 or later (Rust 2024 edition support)
- Root privileges (for most namespace operations, particularly `CLONE_NEWPID` and `CLONE_NEWNS`)

## Project Structure

- `container.go`: Main container implementation with parent and child process logic
- `rust/`: Rust implementation (`container` crate, Rust 2024 edition)
- `container_rootfs/`: Complete Alpine Linux root filesystem used as the container environment
- `bash-container`: Build script or container configuration
- `bash-test-container`: Test script for running containers
- `go.mod`: Go module definition


