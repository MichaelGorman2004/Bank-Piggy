import { spawn } from 'node:child_process';
import { existsSync } from 'node:fs';

if (!existsSync('.env')) throw new Error('Create the private .env file before starting locally.');
process.loadEnvFile('.env');
if (!existsSync('dist/bank-piggy')) throw new Error('Build Bank Piggy with npm run build first.');

const missing = ['PLAID_CLIENT_ID', 'PLAID_SECRET'].filter((name) => !process.env[name]);
if (process.env.PLAID_ENABLED === 'true' && missing.length) {
  console.error(`Add ${missing.join(' and ')} to .env, then run npm run start:local again.`);
  process.exitCode = 1;
} else {
  const app = spawn('./dist/bank-piggy', [], { stdio: 'inherit' });
  app.once('error', () => {
    console.error('The local app could not start. Check the build and private .env settings.');
    process.exitCode = 1;
  });
  app.once('exit', (code) => {
    process.exitCode = code ?? 1;
  });
  process.once('SIGINT', () => {
    app.kill('SIGINT');
  });
  process.once('SIGTERM', () => {
    app.kill('SIGTERM');
  });
}
