use aws_config::Region;
use ini::Ini;
use serde::Deserialize;
use std::process::Command as ProcessCommand;
use std::{collections::HashMap, fs};
use tokio::runtime::Runtime;
use chrono::{DateTime, Utc};

#[derive(Debug)]
pub struct ConfigSection {
    pub name: String,
    profile: ConfigProfile,
}

#[derive(Debug)]
pub struct ConfigProfile {
    region: String,
    sso_account_id: String,
    sso_role_name: String,
    sso_start_url: String,
    sso_region: String,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
struct SsoCache {
    access_token: String,
    expires_at: String,
}

#[derive(Debug)]
pub struct CredentialsSection {
    region: String,
    aws_access_key_id: String,
    aws_secret_access_key: String,
    aws_session_token: String,
}

impl ConfigSection {
    fn sso_cache(&self) -> Option<SsoCache> {
        let mut hasher = sha1_smol::Sha1::new();

        hasher.update(self.profile.sso_start_url.as_bytes());

        let hash = hasher.digest().to_string();
        let cache_path = home::home_dir()?
            .join(".aws/sso/cache")
            .join(hash + ".json");

        let cache: Result<SsoCache, serde_json::Error> = {
            let data = fs::read_to_string(cache_path).expect("awsso: error reading cache file");
            serde_json::from_str(&data)
        };

        cache.ok()
    }
}


pub fn get_credentials() -> HashMap<String, CredentialsSection> {
    let credentials_path = home::home_dir().unwrap().join(".aws/credentials");
    let conf = Ini::load_from_file(credentials_path).unwrap();

    let credentials = conf
        .iter()
        .filter_map(|(sec, prop)| -> Option<(String, CredentialsSection)> {
            let map: HashMap<_, _> = prop.iter().collect();

            let section = CredentialsSection {
                region: map.get("region")?.to_string(),
                aws_access_key_id: map.get("aws_access_key_id")?.to_string(),
                aws_secret_access_key: map.get("aws_secret_access_key")?.to_string(),
                aws_session_token: map.get("aws_session_token")?.to_string(),
            };

            Some((sec?.to_string(), section))
        })
        .collect();

    credentials
}

pub fn write_credentials(sections: &HashMap<String, CredentialsSection>) {
    let credentials_path = home::home_dir().unwrap().join(".aws/credentials");

    let mut conf = Ini::new();

    for (key, credentials) in sections {
        conf.with_section(Some(key.to_string()))
            .set("region", credentials.region.to_string())
            .set("aws_access_key_id", credentials.aws_access_key_id.to_string())
            .set("aws_secret_access_key", credentials.aws_secret_access_key.to_string())
            .set("aws_session_token", credentials.aws_session_token.to_string());
    }

   conf.write_to_file(credentials_path).unwrap();
}

pub fn refresh_creds(profile: &str, login: bool) {
    let config = aws_config();
    let sections: Vec<_> = config
        .values()
        .into_iter()
        .filter(|section| section.name.contains(profile))
        .collect();
    let section = sections.first().unwrap();

    if login {
        ProcessCommand::new("aws")
            .args(["sso", "login", "--profile", &section.name.to_string()])
            .status()
            .expect("awsso: failed to spawn login session");
    }

    let cache = section.sso_cache();
    let rt = Runtime::new().unwrap();

    let creds_result = rt.block_on(async {
        let config = aws_config::defaults(aws_config::BehaviorVersion::latest())
            .region(Region::new(section.profile.region.to_string()))
            .load()
            .await;
        let sso = aws_sdk_sso::Client::new(&config);

        let role_creds = sso
            .get_role_credentials()
            .set_role_name(Some(section.profile.sso_role_name.to_string()))
            .set_account_id(Some(section.profile.sso_account_id.to_string()))
            .set_access_token(match cache {
                None => None,
                Some(c) => { 
                    if let Ok(exp) = DateTime::parse_from_rfc3339(&c.expires_at.to_string()) {
                        assert!(exp.timestamp() > Utc::now().timestamp(), "awsso: sso credentials have expired, please re-run using --login option")
                    }

                    Some(c.access_token)
                },
            })
            .send()
            .await;

        role_creds
    });

    match creds_result {
        Ok(creds) => {
            let mut local_credentials = get_credentials();
            let role_credentials = creds.role_credentials.unwrap();

            local_credentials.insert(
                section.name.to_string(),
                CredentialsSection {
                    region: section.profile.sso_region.to_string(),
                    aws_access_key_id: role_credentials.access_key_id.unwrap().to_string(),
                    aws_secret_access_key: role_credentials.secret_access_key.unwrap().to_string(),
                    aws_session_token: role_credentials.session_token.unwrap().to_string(),
                },
            );

            write_credentials(&local_credentials);

            println!("awsso: credentials were succesfully updated")
        }
        Err(e) => {
            println!("{e:?}")
        }
    }
}

pub fn aws_config() -> HashMap<String, ConfigSection> {
    let config_path = home::home_dir().unwrap().join(".aws/config");
    let conf = Ini::load_from_file(config_path).unwrap();

    let profiles = conf
        .iter()
        .filter_map(|(sec, prop)| -> Option<(String, ConfigSection)> {
            let name = sec?.split_whitespace().last();
            let map: HashMap<_, _> = prop.iter().collect();

            let section = ConfigSection {
                name: name?.to_string(),
                profile: ConfigProfile {
                    region: map.get("region")?.to_string(),
                    sso_account_id: map.get("sso_account_id")?.to_string(),
                    sso_role_name: map.get("sso_role_name")?.to_string(),
                    sso_start_url: map.get("sso_start_url")?.to_string(),
                    sso_region: map.get("sso_region")?.to_string(),
                },
            };

            Some((sec?.to_string(), section))
        })
        .collect();

    profiles
}

