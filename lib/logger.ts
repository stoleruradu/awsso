import chalk, { ForegroundColor } from 'chalk';

export class Logger {
    warn(message: string): void {
        this.log(message, 'yellow');
    }

    error(message: string): void {
        this.log(message, 'red');
    }

    success(message: string): void {
        this.log(message, 'green');
    }

    info(message: string): void {
        this.log(message, 'cyan');
    }

    private log(message: string, color: typeof ForegroundColor): void {
        console.log(chalk[color](message));
    }
}

export default new Logger();
