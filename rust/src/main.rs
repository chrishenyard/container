use clap::{Parser, Subcommand, ValueEnum};
use nix::mount::{mount, umount};
use nix::sched::{unshare, CloneFlags};
use nix::unistd::{chdir, chroot, getpid, getuid, sethostname};
use std::fs;
use std::process::{self, Command as ProcessCommand, Stdio};

#[derive(Clone, Copy, Debug, Eq, PartialEq, ValueEnum)]
enum Mode {
    Root,
    User,
}

impl Mode {
    fn as_str(self) -> &'static str {
        match self {
            Mode::Root => "root",
            Mode::User => "user",
        }
    }
}

#[derive(Debug, Subcommand)]
enum Command {
    Run {
        #[arg(required = true, trailing_var_arg = true)]
        command: Vec<String>,
    },
    #[command(hide = true)]
    Child {
        #[arg(required = true, trailing_var_arg = true)]
        command: Vec<String>,
    },
}

#[derive(Debug, Parser)]
#[command(author, version, about, long_about = None)]
struct Cli {
    #[arg(long, value_enum, default_value_t = Mode::Root)]
    mode: Mode,

    #[command(subcommand)]
    command: Command,
}

fn main() {
    let cli = Cli::parse();

    match cli.command {
        Command::Run { command } => parent(cli.mode, command),
        Command::Child { command } => child(cli.mode, command),
    }
}

fn parent(mode: Mode, command: Vec<String>) {
    let uid = getuid();

    println!("Current User ID (Integer): {}", uid);
    if uid.as_raw() == 0 {
        println!("Running with root privileges.");
    }
    println!("Running parent process (PID {})", getpid());

    let mut clone_flags = CloneFlags::CLONE_NEWUTS | CloneFlags::CLONE_NEWPID | CloneFlags::CLONE_NEWNS;
    if mode == Mode::User {
        clone_flags |= CloneFlags::CLONE_NEWUSER;
    }

    if let Err(err) = unshare(clone_flags) {
        eprintln!("Error creating namespaces in parent: {err}");
        process::exit(1);
    }

    if mode == Mode::User {
        if let Err(err) = configure_user_namespace(uid.as_raw(), nix::unistd::getgid().as_raw()) {
            eprintln!("Error configuring user namespace: {err}");
            process::exit(1);
        }
    }

    let mut args = vec![format!("--mode={}", mode.as_str()), String::from("child")];
    args.extend(command);

    let status = ProcessCommand::new("/proc/self/exe")
        .args(args)
        .stdin(Stdio::inherit())
        .stdout(Stdio::inherit())
        .stderr(Stdio::inherit())
        .status();

    match status {
        Ok(status) if status.success() => {}
        Ok(status) => {
            eprintln!("Child process failed with status: {status}");
            process::exit(status.code().unwrap_or(1));
        }
        Err(err) => {
            eprintln!("Error running parent: {err}");
            process::exit(1);
        }
    }
}

fn child(_mode: Mode, command: Vec<String>) {
    println!("Running child process (PID {} inside container)", getpid());

    let uid = getuid();
    println!("Current User ID (Integer): {}", uid);
    if uid.as_raw() == 0 {
        println!("Running with root privileges.");
    }

    if let Err(err) = sethostname("isolated-container") {
        eprintln!("Error setting hostname: {err}");
        process::exit(1);
    }

    if let Err(err) = chroot("container_rootfs/") {
        eprintln!("Error chrooting to container_rootfs/: {err}");
        process::exit(1);
    }

    if let Err(err) = chdir("/") {
        eprintln!("Error changing directory to /: {err}");
        process::exit(1);
    }

    if let Err(err) = mount(Some("proc"), "/proc", Some("proc"), nix::mount::MsFlags::empty(), None::<&str>) {
        eprintln!("Error mounting /proc: {err}");
        process::exit(1);
    }

    let status = ProcessCommand::new(&command[0])
        .args(&command[1..])
        .stdin(Stdio::inherit())
        .stdout(Stdio::inherit())
        .stderr(Stdio::inherit())
        .status();

    if let Err(err) = umount("/proc") {
        eprintln!("Error unmounting /proc: {err}");
    }

    match status {
        Ok(status) if status.success() => {}
        Ok(status) => {
            eprintln!("Error running child command: process exited with status {status}");
            process::exit(status.code().unwrap_or(1));
        }
        Err(err) => {
            eprintln!("Error running child: {err}");
            process::exit(1);
        }
    }
}

fn configure_user_namespace(host_uid: u32, host_gid: u32) -> Result<(), std::io::Error> {
    // The kernel requires disabling setgroups before writing gid_map for unprivileged mappings.
    fs::write("/proc/self/setgroups", "deny")?;
    fs::write("/proc/self/uid_map", format!("0 {host_uid} 1\n"))?;
    fs::write("/proc/self/gid_map", format!("0 {host_gid} 1\n"))?;
    Ok(())
}
