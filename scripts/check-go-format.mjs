import { execFileSync } from 'node:child_process';

try {
  const unformatted = execFileSync('gofmt', ['-l', 'apps/api'], { encoding: 'utf8' }).trim();
  if (unformatted) {
    console.error(`Go formatting needs correction:\n${unformatted}`);
    process.exitCode = 1;
  }
} catch (error) {
  console.error(`Go format check failed: ${error.message}`);
  process.exitCode = 1;
}
