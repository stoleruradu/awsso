import { CLI } from '../lib/cli.js';
import { Logger } from '../lib/logger.js';

if (!process.argv.slice(2).length) {
    CLI.outputHelp();
} else {
    CLI.parseAsync(process.argv)
        .catch((error: Error) => Logger.error(error.message));
}
