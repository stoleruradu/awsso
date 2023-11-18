import { createCommand } from 'commander';
import { CredsCommandOptions, listProfiles, updateShortTermCredentials } from './aws.js';
import pkg from '../package.json' assert { type: 'json' };

const CLI = createCommand(pkg.name)
    .version(pkg.version)
    .usage('<command> [options]')
    .helpOption('-h, --help', 'output usage information');

CLI.command('creds <profiles...>')
    .usage('<profiles...> [options]')
    .option('--backup', 'Makes a buckup before writing to credentials file')
    .option('--dryRun', 'Writes to stdout')
    .option('--login', 'Creates an AWS SSO login session before fetching credentials')
    .description('Refresh short-term credentials')
    .action(async (profiles: string[], options: CredsCommandOptions) => updateShortTermCredentials(profiles, options));

CLI.command('profiles')
    .description('List available sso profiles')
    .action(() => listProfiles());


export { CLI };
export default CLI;
