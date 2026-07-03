package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	var (
		mode   string
		rootfs string
	)

	flag.StringVar(&mode, "mode", "root", "Mode of operation: root or user")
	flag.StringVar(&rootfs, "rootfs", "", "Path to the container root filesystem")
	flag.Parse()

	if rootfs == "" {
		fmt.Println("Missing required --rootfs argument")
		os.Exit(1)
	}

	args := flag.Args()

	if len(args) < 2 {
		fmt.Println("Usage: <go run container.go | container> --mode=<root|user> run <command>")
		os.Exit(1)
	}

	if mode != "root" && mode != "user" {
		fmt.Println("Invalid mode. Use 'root' or 'user'.")
		os.Exit(1)
	}

	switch args[0] {
	case "run":
		parent(mode, rootfs, args[1:])
	case "child":
		child(rootfs, args[1:])
	default:
		panic("Unknown command")
	}
}

// First execution: Request new namespaces from the Linux Kernel
func parent(mode, rootfs string, commandArgs []string) {
	uid := os.Getuid()

	fmt.Printf("Current User ID (Integer): %d\n", uid)

	if uid == 0 {
		fmt.Println("Running with root privileges.")
	}

	fmt.Printf("Running parent process (PID %d)\n", os.Getpid())

	// Re-execute this same binary, but call the 'child' case.
	args := []string{
		"--mode=" + mode,
		"--rootfs=" + rootfs,
		"child",
	}
	args = append(args, commandArgs...)

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
			Cloneflags: syscall.CLONE_NEWUTS |
				syscall.CLONE_NEWPID |
				syscall.CLONE_NEWNS |
				syscall.CLONE_NEWUSER,

			UidMappings: []syscall.SysProcIDMap{
				{
					ContainerID: 0,
					HostID:      os.Getuid(),
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
func child(rootfs string, commandArgs []string) {
	if len(commandArgs) == 0 {
		fmt.Println("Missing command.")
		os.Exit(1)
	}

	fmt.Printf("Running child process (PID %d inside container)\n", os.Getpid())

	uid := os.Getuid()

	fmt.Printf("Current User ID (Integer): %d\n", uid)

	if uid == 0 {
		fmt.Println("Running with root privileges.")
	}

	must(syscall.Sethostname([]byte("isolated-container")))

	must(syscall.Chroot(rootfs))
	must(os.Chdir("/"))

	must(syscall.Mount("proc", "proc", "proc", 0, ""))

	cmd := exec.Command(commandArgs[0], commandArgs[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running child: %v\n", err)
	}

	syscall.Unmount("/proc", 0)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
