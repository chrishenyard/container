use clap::{Parser, Subcommand, ValueEnum};
use std::process::{self, Command as ProcessCommand, Stdio};

#[derive(Clone, Copy, Debug, Eq, PartialEq, ValueEnum)]
enum Mode {
    Root,
    User,
}

#[derive(Debug, Subcommand)]
enum Command {
    Run {
        #[arg(required = true, trailing_var_arg = true)]
        command: Vec<String>,
    },
}

#[derive(Debug, Parser)]
#[command(author, version, about, long_about = None)]
struct Cli {
    #[arg(long, value_enum, default_value_t = Mode::Root)]
    mode: Mode,

    #[arg(long)]
    rootfs: String,

    #[command(subcommand)]
    command: Command,
}

fn main() {
    let cli = Cli::parse();

    match cli.command {
        Command::Run { command } => run(cli.mode, cli.rootfs, command),
    }
}

fn run(mode: Mode, rootfs: String, command: Vec<String>) {
    if command.is_empty() {
        eprintln!("Missing command.");
        process::exit(1);
    }

    let mut args = vec![
        "--uts".to_string(),
        "--pid".to_string(),
        "--mount".to_string(),
        "--fork".to_string(),
        "--mount-proc".to_string(),
        "--root".to_string(),
        rootfs,
        "--wd".to_string(),
        "/".to_string(),
    ];

    if mode == Mode::User {
        args.insert(0, "--map-root-user".to_string());
        args.insert(0, "--user".to_string());
    }

    args.push("--".to_string());
    args.extend(command);

    let status = ProcessCommand::new("unshare")
        .args(args)
        .stdin(Stdio::inherit())
        .stdout(Stdio::inherit())
        .stderr(Stdio::inherit())
        .status();

    match status {
        Ok(status) if status.success() => {}

        Ok(status) => {
            eprintln!("Container exited with status: {status}");
            process::exit(status.code().unwrap_or(1));
        }

        Err(err) => {
            eprintln!("Error running unshare: {err}");
            process::exit(1);
        }
    }
}
