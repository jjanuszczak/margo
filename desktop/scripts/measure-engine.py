#!/usr/bin/env python3
"""Measure protocol latency without changing authored project files."""
import json
import pathlib
import statistics
import subprocess
import time

root = pathlib.Path(__file__).resolve().parents[2]
engine = next((root / 'desktop/src-tauri/binaries').glob('margo-engine-*'))
started = time.perf_counter()
process = subprocess.Popen([str(engine), 'desktop-engine'], stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)
def call(op, **params):
    begin = time.perf_counter()
    process.stdin.write(json.dumps(dict(protocol=1, op=op, **params))+'\n')
    process.stdin.flush()
    response = json.loads(process.stdout.readline())
    if response.get('error'):
        raise RuntimeError(response['error'])
    return response, round((time.perf_counter()-begin)*1000, 1)
try:
    call('hello')
    print(f'Engine cold handshake: {(time.perf_counter()-started)*1000:.1f}ms')
    for name in ['reference-deck', 'benchmark-deck', 'omarchy-theme-showcase', 'arca-investor-memo']:
        path = root / 'examples' / name
        if not path.exists():
            continue
        opened, latency = call('open', root=str(path))
        slide = opened['snapshot']['slides'][0]['id']
        samples = [call('preview', slide=slide)[1] for _ in range(5)]
        print(f'{name}: open {latency}ms, preview median {statistics.median(samples)}ms, max {max(samples)}ms')
finally:
    process.stdin.close()
    process.wait(timeout=10)
