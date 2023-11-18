import AWS from 'aws-sdk';
import { spawn } from 'child_process';
import crypto from 'crypto';
import fs from 'fs-extra';
import * as ConfigParser from 'ini';
import os, { EOL } from 'os';
import { Logger } from './logger.js';

export type CredsCommandOptions = {
    dryRun?: boolean;
    backup?: boolean;
    login?: boolean;
}

type Profile = {
    name: string;
    raw: {
        region: string;
        output: string;
        sso_account_id: string;
        sso_role_name: string;
        sso_start_url: string;
        sso_region: string;
    }
}

type CacheCredentials = {
    startUrl: string;
    region: string;
    accessToken: string;
    expiresAt: string;
}

type WriteConfigOptions<T> = {
    path: string;
    data: T;
    dryRun?: boolean;
    backup?: boolean;
}

const HOME_PATH = os.homedir();
const AWS_CONFIG_PATH = `${HOME_PATH}/.aws/config`;
const AWS_CREDENTIAL_PATH = `${HOME_PATH}/.aws/credentials`;
const AWS_SSO_CACHE_PATH = `${HOME_PATH}/.aws/sso/cache`;

const opneLoginSession = async (profile: Profile): Promise<void> => {
    const login = spawn('aws sso login', [`--profile ${profile.name}`], { shell: true });

    return new Promise((resolve, reject) => {
        login.stdout.on("data", data => {
            Logger.info(data);
        });

        login.stderr.on("data", data => {
            Logger.error(data);
        });

        login.on('error', reject);
        login.on('close', resolve);

    });
}

const readConfig = async <T = Record<string, any>>(path: string): Promise<T> => {
    const configFile = await fs.readFile(path);

    return ConfigParser.parse(configFile.toString()) as T;
}

const writeConfig = async <T = Record<string, any>>({ data, path, dryRun = false, backup = false }: WriteConfigOptions<T>): Promise<void> => {
    Logger.info(`Updating credential files`);
    const file = ConfigParser.stringify(data, { whitespace: false });

    if (dryRun) {
        Logger.info(`Found dry-run flag, writing to stdout...${EOL}${file}`);
        return;
    }

    if (backup) {
        const bakFile = fs.readFileSync(path);
        const bakPath = `${path}.bak`;

        Logger.info(`Making buckup ${path} => ${bakPath}`);
        await fs.writeFile(bakPath, bakFile);
    }

    await fs.writeFile(path, file);
}

const getProfile = async (name: string): Promise<Profile> => {
    Logger.info(`Reading profile: ${name}`);

    const config = await readConfig(AWS_CONFIG_PATH);
    const [_, raw] =  Object
        .entries(config)
        .find(([key]) => key.includes(name)) ?? [];

    if (!raw) {
        throw new Error(`AWS profile [${name}] was not found. Check credentials file...`);
    }

    return { name, raw };
}

const getCachedCreds = async (profile: Profile, login: boolean = false): Promise<CacheCredentials> => {
    if (login) {
        await opneLoginSession(profile);
    }

    Logger.info(`Checking for SSO credentials...`);

    const cacheHash = crypto
        .createHash('sha1')
        .update(profile.raw.sso_start_url)
        .digest('hex');

    const [error, buffer] = await fs.readFile(`${AWS_SSO_CACHE_PATH}/${cacheHash}.json`)
        .then((value) => [, value], (reason) => [reason]);

    if (error instanceof Error || !Buffer.isBuffer(buffer)) {
        const message = [
          'Missing SSO credentials.',
          `Please re-validate by running: aws sso login --profile ${profile.name}`,
        ].join(EOL);

        throw new Error(message);
    }

    const config = JSON.parse(buffer.toString()) as CacheCredentials;
    const now = Date.now();

    if (profile.raw.sso_region !== config.region) {
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

    const sso = new AWS.SSO({ region: profile.raw.sso_region });

    const [error, credentials] = await sso
        .getRoleCredentials({
            roleName: profile.raw.sso_role_name,
            accountId: profile.raw.sso_account_id,
            accessToken: cache.accessToken,
        })
        .promise()
        .then((value) => [, value], (reason) => [reason]);

    if (error instanceof Error) {
        throw error;
    }

    if (!credentials?.roleCredentials) {
        throw new Error('Failed to get short-term credentials...');
    }

    return credentials.roleCredentials;
}

export const updateShortTermCredentials = async (profiles: string[], options: CredsCommandOptions): Promise<void> => {
    const startTime = Date.now();
    const config = await readConfig(AWS_CREDENTIAL_PATH);

    for (const profileName of profiles) {
        const profile = await getProfile(profileName);
        const cache = await getCachedCreds(profile, options.login);
        const credentials = await getRoleCredentials(profile, cache);

        config[profileName] = {
            region: cache.region,
            aws_access_key_id: credentials.accessKeyId,
            aws_secret_access_key: credentials.secretAccessKey,
            aws_session_token: credentials.sessionToken,
        }
    }

    await writeConfig({ data: config, path: AWS_CREDENTIAL_PATH, ...options });

    Logger.success(`Done in ${Date.now() - startTime} ms`);
}

export const listProfiles = async (): Promise<void> => {
    const config = await readConfig(AWS_CONFIG_PATH);

    const profiles = Object
        .keys(config)
        .filter((key) => !!config[key].sso_start_url)
        .map((key) => {
            const [_, profileName] = key.split('profile ');
            return `${config[key].sso_account_id} ${profileName}`;
        });

    const acountIdHeader = 'ACCOUNT ID';

    const [first] = profiles;
    const [accountId] = first.split(' ');

    const spaces = new Array(accountId.length - acountIdHeader.length).fill(' ').join('');
    const header = `${acountIdHeader + spaces} PROFILE NAME`;

    Logger.info([header, ...profiles].join(EOL));
}

