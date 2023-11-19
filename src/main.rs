use clap::Command;
use ini::Ini;
use std::collections::HashMap;

#[derive(Debug)]
struct ConfigProfile {
    region: String,
    output: String,
    sso_account_id: String,
    sso_role_name: String,
    sso_start_url: String,
    sso_region: String,
}

#[derive(Debug)]
struct ConfigSection {
    name: String,
    profile: ConfigProfile,
}

fn config_sections() -> Vec<ConfigSection> {
    let home_path = home::home_dir().unwrap().to_str().unwrap().to_string();
    let config_path = home_path + "/.aws/config";
    let conf = Ini::load_from_file(config_path).unwrap();

    let profiles = conf.iter().filter_map(|(sec, prop)| -> Option<ConfigSection> {
        let name = sec?.split_whitespace().last();
        let map: HashMap<_, _> = prop.iter().collect();

        let section = ConfigSection {
            name: name?.to_string(),
            profile: ConfigProfile {
                region: map.get("region")?.to_string(),
                output: map.get("output")?.to_string(),
                sso_account_id: map.get("sso_account_id")?.to_string(),
                sso_role_name: map.get("sso_role_name")?.to_string(),
                sso_start_url: map.get("sso_start_url")?.to_string(),
                sso_region: map.get("sso_region")?.to_string(),
            },
        };

        Some(section)
    }).collect();

    profiles
}

fn main() {
    let matches = Command::new("awssso")
        .about("AWS sso helper")
        .version("0.0.1")
        .subcommand_required(true)
        .arg_required_else_help(true)
        .subcommand(Command::new("profiles").about("List the available profiles"))
        .get_matches();

    match matches.subcommand() {
        Some(("profiles", _)) => {
            let sections = config_sections();

            for section in sections.iter() {
                println!("{:#?}", section);
            }
        }
        _ => unreachable!(),
    }
}
