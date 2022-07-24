import * as AWS from 'aws-sdk';
import os, { EOL } from 'os';
import * as ConfigParser from 'ini';
import fs from 'fs';
import crypto from 'crypto';
import Logger from './logger';

export type CredsCommandOptions = {
    readonly dryRun?: boolean;
    readonly backup?: boolean;
}

type Profile = {
    readonly region: string;
    readonly output: string;
    readonly sso_account_id: string;
    readonly sso_role_name: string;
    readonly sso_start_url: string;
    readonly sso_region: string;
}

type CacheCredentials = {
    readonly startUrl: string;
    readonly region: string;
    readonly accessToken: string;
    readonly expiresAt: string;
}

type WriteConfigOptions<T> = {
    readonly path: string;
    readonly data: T;
    readonly dryRun?: boolean;
    readonly backup?: boolean;
}

const HOME_PATH = os.homedir();
const AWS_CONFIG_PATH = `${HOME_PATH}/.aws/config`;
const AWS_CREDENTIAL_PATH = `${HOME_PATH}/.aws/credentials`;
const AWS_SSO_CACHE_PATH = `${HOME_PATH}/.aws/sso/cache`;

const readConfig = <T = Record<string, any>>(path: string): T => {
    const configFile = fs.readFileSync(path).toString();

    return ConfigParser.parse(configFile) as T;
}

const writeConfig = <T = Record<string, any>>({ data, path, dryRun = false, backup = false }: WriteConfigOptions<T>): void => {
    Logger.info(`Updating credential files`);
    const file = ConfigParser.stringify(data, { whitespace: true });

    if (dryRun) {
        Logger.info(`Found dry-run flag, writing to stdout...${EOL}${file}`);
        return;
    }

    if (backup) {
        const bakFile = fs.readFileSync(path);
        const bakPath = `${path}.bak`;

        Logger.info(`Making buckup ${path} => ${bakPath}`);
        fs.writeFileSync(bakPath, bakFile);
    }

    fs.writeFileSync(path, file);
}

const getProfile = (profileName: string): Profile => {
    Logger.info(`Reading profile: ${profileName}`);

    const config = readConfig(AWS_CONFIG_PATH);
    const [_, profile] =  Object
        .entries(config)
        .find(([key]) => key.includes(profileName)) ?? [];

    if (!profile) {
        throw new Error(`AWS profile [${profileName}] was not found. Check credentials file...`);
    }

    return profile as Profile;
}

const getCachedCreds = (profile: Profile): CacheCredentials => {
    Logger.info(`Checking for SSO credentials...`);
    const cacheHash = crypto
        .createHash('sha1')
        .update(profile.sso_start_url)
        .digest('hex');

    const cacheBuffer = fs.readFileSync(`${AWS_SSO_CACHE_PATH}/${cacheHash}.json`);
    const config = JSON.parse(cacheBuffer.toString()) as CacheCredentials;

    const now = Date.now();

    if (profile.sso_region !== config.region) {
        throw new Error('SSO authentication region in cache does not match region defined in profile');
    }

    const expiresAt = new Date(config.expiresAt);

    if (now > expiresAt.getTime()) {
        throw new Error('SSO credentials have expired. Please re-validate with the AWS CLI tool or --login option.');
    }

    return config;
}

const getRoleCredentials = async (profile: Profile, cache: CacheCredentials): Promise<AWS.SSO.RoleCredentials> => {
    Logger.info('Fetching short-term CLI session token...');

    const sso = new AWS.SSO({ region: profile.sso_region });

    const credentials = await sso
        .getRoleCredentials({
            roleName: profile.sso_role_name,
            accountId: profile.sso_account_id,
            accessToken: cache.accessToken,
        })
        .promise();

    if (!credentials.roleCredentials) {
        throw new Error('Failed to get short-term credentials...');
    }

    return credentials.roleCredentials;
}

export const updateShortTermCredentials = async (profiles: string[], options: CredsCommandOptions): Promise<void> => {
    const startTime = Date.now();
    const config = readConfig(AWS_CREDENTIAL_PATH);

    for (const profileName of profiles) {
        const profile = getProfile(profileName);
        const cache = getCachedCreds(profile);
        const credentials = await getRoleCredentials(profile, cache);

        config[profileName] = {
            region: cache.region,
            aws_access_key_id: credentials.accessKeyId,
            aws_secret_access_key: credentials.secretAccessKey,
            aws_session_token: credentials.sessionToken,
        }
    }

    writeConfig({ data: config, path: AWS_CREDENTIAL_PATH, ...options });

    Logger.success(`Done in ${Date.now() - startTime} ms`);
}

export const listProfiles = (): void => {
    const config = readConfig(AWS_CONFIG_PATH);

    const profiles = Object
        .keys(config)
        .filter((key) => {
            if (config[key].sso_start_url) {
                return true;
            }

            return false;
        })
        .map((key) => {
            const [_, profileName] = key.split('profile ');

            return profileName;
        })
        .join(EOL);

    Logger.info(profiles);
}

