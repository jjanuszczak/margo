import { execFileSync } from 'node:child_process';
import { mkdirSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
const root = fileURLToPath(new URL('../../', import.meta.url));
const triple = process.env.TAURI_ENV_TARGET_TRIPLE || execFileSync('rustc', ['-vV'], { encoding: 'utf8' }).match(/^host: (.+)$/m)[1];
const targets = {
  'aarch64-apple-darwin': ['darwin', 'arm64'],
  'x86_64-apple-darwin': ['darwin', 'amd64'],
  'x86_64-unknown-linux-gnu': ['linux', 'amd64'],
  'aarch64-unknown-linux-gnu': ['linux', 'arm64'],
};
if (!targets[triple]) throw new Error(`Unsupported desktop target: ${triple}`);
mkdirSync(path.join(root, 'desktop/src-tauri/binaries'), { recursive: true });
const args = ['build', '-trimpath'];
if (process.env.MARGO_VERSION) {
  const version = process.env.MARGO_VERSION.replace(/^v/, '');
  args.push('-ldflags', `-s -w -X github.com/jjanuszczak/margo/internal/version.Version=${version}`);
}
args.push('-o', `desktop/src-tauri/binaries/margo-engine-${triple}`, './cmd/margo');
execFileSync('go', args, {
  cwd: root, stdio: 'inherit', env: { ...process.env, GOOS: targets[triple][0], GOARCH: targets[triple][1], CGO_ENABLED: '0', GOCACHE: path.join(root, '.gocache'), GOMODCACHE: path.join(root, '.gomodcache') },
});
