import { createCommand } from 'commander';
import { CredsCommandOptions, listProfiles, updateShortTermCredentials } from './aws-sso';
import pkg from '../package.json';

const CLI = createCommand(pkg.name)
    .version(pkg.version)
    .usage('<command> [options]')
    .helpOption('-h, --help', 'output usage information');

CLI.command('creds <profiles...>')
    .usage('<profiles...> [options]')
    .option('--dryRun', 'Writing to stdout')
    .option('--backup', 'Make a buckup before writing to credentials file')
    .description('Refresh short-term credentials')
    .action(async (profiles: string[], options: CredsCommandOptions) => updateShortTermCredentials(profiles, options));

CLI.command('profiles')
    .description('List available sso profiles')
    .action(() => listProfiles());


export { CLI };
export default CLI;
