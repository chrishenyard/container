package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: <go run container.go | container> --mode=<root|user> run <command>")
		os.Exit(1)
	}

	var mode string
	flag.StringVar(&mode, "mode", "root", "Mode of operation: root or user")
	flag.Parse()

	if mode != "root" && mode != "user" {
		fmt.Println("Invalid mode. Use 'root' or 'user'.")
		os.Exit(1)
	}

	switch os.Args[2] {
	case "run":
		parent(mode)
	case "child":
		child()
	default:
		panic("Unknown command")
	}
}

// First execution: Request new namespaces from the Linux Kernel
func parent(mode string) {
	uid := os.Getuid()

	fmt.Printf("Current User ID (Integer): %d\n", uid)

	// Example: Check if the program is running as root
	if uid == 0 {
		fmt.Println("Running with root privileges.")
	}

	fmt.Printf("Running parent process (PID %d)\n", os.Getpid())

	// Re-execute this same binary, but call the 'child' case
	// Get args
	var args []string = []string{os.Args[1], "child"}
	args = append(args, os.Args[3:]...)

	cmd := exec.Command("/proc/self/exe", args...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	switch mode {
	case "root":
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		}
	case "user":
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWUSER,
			UidMappings: []syscall.SysProcIDMap{
				{
					ContainerID: 0,           // Becomes root (UID 0) inside container
					HostID:      os.Getuid(), // Maps from your current host user ID
					Size:        1,
				},
			},
			GidMappings: []syscall.SysProcIDMap{
				{
					ContainerID: 0,
					HostID:      os.Getgid(),
					Size:        1,
				},
			},
		}
	}

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running parent: %v\n", err)
		os.Exit(1)
	}
}

// Second execution: Runs inside the newly created namespaces
func child() {
	fmt.Printf("Running child process (PID %d inside container)\n", os.Getpid())

	uid := os.Getuid()

	fmt.Printf("Current User ID (Integer): %d\n", uid)

	// Example: Check if the program is running as root
	if uid == 0 {
		fmt.Println("Running with root privileges.")
	}

	// 1. Isolate the Hostname
	syscall.Sethostname([]byte("isolated-container"))

	// 2. Chroot into our Alpine directory
	must(syscall.Chroot("container_rootfs/")) // <-- CHANGE TO YOUR ACTUAL PATH
	must(os.Chdir("/"))

	// 3. Mount the isolated /proc filesystem so 'ps' works correctly
	must(syscall.Mount("proc", "proc", "proc", 0, ""))

	// 4. Run the user's requested command
	cmd := exec.Command(os.Args[3], os.Args[4:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running child: %v\n", err)
	}

	// Clean up the mount when the container exits
	syscall.Unmount("/proc", 0)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
