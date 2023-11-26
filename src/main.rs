mod aws;

use crate::aws::refresh_creds;
use crate::aws::aws_config;

use clap::crate_version;
use clap::{Arg, ArgAction, Command};

fn main() {
    let cli = Command::new("awsso")
        .about("AWS sso helper")
        .version(crate_version!())
        .subcommand_required(true)
        .arg_required_else_help(true)
        .subcommand(Command::new("profiles").about("List available sso profiles"))
        .subcommand(
            Command::new("creds")
                .about("Refresh short-term credentials")
                .arg(Arg::new("profile").required(true).action(ArgAction::Set))
                .arg(Arg::new("login").long("login").action(ArgAction::SetTrue)),
        )
        .get_matches();

    match cli.subcommand() {
        Some(("profiles", _)) => {
            let sections = aws_config();

            for section in sections.values() {
                println!("{}", section.name);
            }
        }
        Some(("creds", creds_matches)) => {
            let profile = creds_matches.get_one::<String>("profile");
            let login = creds_matches.get_flag("login");

            refresh_creds(profile.unwrap(), login);
        }
        _ => unreachable!(),
    }
}
