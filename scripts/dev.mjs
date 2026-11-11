import { spawn } from 'node:child_process';
import { existsSync } from 'node:fs';

if (existsSync('.env')) process.loadEnvFile('.env');

const developmentEnvironment = {
  ...process.env,
  APP_MODE: process.env.APP_MODE ?? 'demo',
  LISTEN_ADDR: process.env.LISTEN_ADDR ?? '127.0.0.1:8080',
  APP_BASE_URL: process.env.APP_BASE_URL ?? 'http://127.0.0.1:5173',
};
const useProcessGroups = process.platform !== 'win32';

const processes = [
  spawn('go', ['run', './cmd/server'], {
    cwd: 'apps/api',
    env: developmentEnvironment,
    stdio: 'inherit',
    detached: useProcessGroups,
  }),
  spawn('npm', ['run', 'dev:web'], {
    env: developmentEnvironment,
    stdio: 'inherit',
    detached: useProcessGroups,
  }),
];
let isStopping = false;

/** Stops both development processes when either exits or the terminal closes. */
function stopProcesses() {
  if (isStopping) return;
  isStopping = true;
  for (const child of processes) terminateProcess(child);
}

/** Releases the process group, including Go and npm child servers. */
function terminateProcess(child) {
  if (!child.pid) return;
  try {
    if (useProcessGroups) process.kill(-child.pid, 'SIGTERM');
    else child.kill('SIGTERM');
  } catch (error) {
    if (error.code !== 'ESRCH') {
      console.error(`Could not stop a development process: ${error.message}`);
      process.exitCode ||= 1;
    }
  }
}

for (const child of processes) {
  child.on('error', (error) => {
    console.error(error.message);
    process.exitCode = 1;
    stopProcesses();
  });
  child.on('exit', (code) => {
    if (code && !isStopping) process.exitCode = code;
    stopProcesses();
  });
}
process.on('SIGINT', stopProcesses);
process.on('SIGTERM', stopProcesses);
