import assert from 'node:assert/strict';
import { jsonSchema, stepCountIs, streamText, tool } from 'ai';
import { createOpenAI } from '@ai-sdk/openai';
import { createAnthropic } from '@ai-sdk/anthropic';
import { createGoogleGenerativeAI } from '@ai-sdk/google';
import { createNativeOpenAIFetch } from '../src/ai-sdk-native-fetch.js';

const nativeCredentialPlaceholder = 'margo-native-credential-placeholder';
const observed = [];
const subscriptions = new Map();
const mockListen = async (name, callback) => {
  subscriptions.set(name, callback);
  return () => subscriptions.delete(name);
};
const emit = (name, payload) => subscriptions.get(name)?.({ payload });
const mockInvoke = async (command, args) => {
  assert.equal(command, 'ai_sdk_proxy_start');
  assert.equal(args.authMode, 'chatgpt');
  assert.equal(args.sessionOnly, true);
  observed.push({ ...args, payload: JSON.parse(args.body) });
  const events = [];
  if (observed.length === 1) {
    const item = { id: 'fc_probe', type: 'function_call', status: 'completed', name: 'readProjectFile', arguments: '{"path":"themes/current/theme.yaml"}', call_id: 'call_probe' };
    events.push(
      ['response.created', { type: 'response.created', response: { id: 'resp_tool', object: 'response', status: 'in_progress', model: 'margo-spike' } }],
      ['response.output_item.added', { type: 'response.output_item.added', output_index: 0, item: { ...item, status: 'in_progress', arguments: '' } }],
      ['response.function_call_arguments.done', { type: 'response.function_call_arguments.done', output_index: 0, item_id: item.id, arguments: item.arguments }],
      ['response.output_item.done', { type: 'response.output_item.done', output_index: 0, item }],
      ['response.completed', { type: 'response.completed', response: { id: 'resp_tool', object: 'response', status: 'completed', model: 'margo-spike', output: [item], usage: { input_tokens: 8, output_tokens: 3, total_tokens: 11 } } }],
    );
  } else {
    const message = { id: 'msg_probe', type: 'message', role: 'assistant', status: 'completed', content: [{ type: 'output_text', text: 'The active theme declares nested tiling support.' }] };
    events.push(
      ['response.created', { type: 'response.created', response: { id: 'resp_text', object: 'response', status: 'in_progress', model: 'margo-spike' } }],
      ['response.output_item.added', { type: 'response.output_item.added', output_index: 0, item: { ...message, status: 'in_progress', content: [] } }],
      ['response.output_text.delta', { type: 'response.output_text.delta', item_id: message.id, output_index: 0, content_index: 0, delta: message.content[0].text }],
      ['response.output_item.done', { type: 'response.output_item.done', output_index: 0, item: message }],
      ['response.completed', { type: 'response.completed', response: { id: 'resp_text', object: 'response', status: 'completed', model: 'margo-spike', output: [message], usage: { input_tokens: 15, output_tokens: 9, total_tokens: 24 } } }],
    );
  }
  const encoder = new TextEncoder();
  queueMicrotask(() => {
    emit(`margo-ai-sdk-${args.requestId}`, { kind: 'headers', status: 200, headers: { 'content-type': 'text/event-stream' } });
    for (const [index, [name, data]] of events.entries()) {
      data.sequence_number = index + 1;
      if (data.response) data.response.created_at ??= 1_700_000_000;
      const frame = encoder.encode(`event: ${name}\ndata: ${JSON.stringify(data)}\n\n`);
      const base64 = btoa(String.fromCharCode(...frame));
      emit(`margo-ai-sdk-${args.requestId}`, { kind: 'chunk', base64 });
    }
    emit(`margo-ai-sdk-${args.requestId}`, { kind: 'done' });
  });
};

const openaiFetch = createNativeOpenAIFetch({
  invoke: mockInvoke,
  listen: mockListen,
  authMode: 'chatgpt',
  sessionOnly: true,
});
const openai = createOpenAI({ apiKey: nativeCredentialPlaceholder, fetch: openaiFetch });
const anthropic = createAnthropic({ apiKey: 'spike-only' });
const google = createGoogleGenerativeAI({ apiKey: 'spike-only' });
assert.equal(typeof anthropic, 'function');
assert.equal(typeof google, 'function');

let toolRuns = 0;
const result = streamText({
    model: openai.responses('margo-spike'),
    prompt: 'Check whether the active theme supports nested tiling.',
    tools: {
      readProjectFile: tool({
        description: 'Read a project text file.',
        inputSchema: jsonSchema({
          type: 'object',
          properties: { path: { type: 'string' } },
          required: ['path'],
          additionalProperties: false,
        }),
        execute: async ({ path }) => {
          toolRuns++;
          assert.equal(path, 'themes/current/theme.yaml');
          return { path, content: 'features:\n  nested_tiles: true\n' };
        },
      }),
    },
    stopWhen: stepCountIs(3),
  });
const text = await result.text;
const steps = await result.steps;
assert.match(text, /declares nested tiling support/);
assert.equal(toolRuns, 1);
assert.equal(steps.length, 2);
assert.ok(observed.every(request => !Object.hasOwn(request, 'authorization')));
assert.ok(observed.every(request => !Object.hasOwn(request, 'headers')));
assert.ok(observed.every(request => request.url === 'https://api.openai.com/v1/responses'));
assert.ok(!JSON.stringify(observed).includes('real-secret'));

console.log('PASS: OpenAI Responses streaming, one read-only tool round-trip, and final text.');
console.log('PASS: OpenAI, Anthropic, and Google provider packages import.');
console.log('PASS: custom fetch routes Responses SSE through a mocked Tauri event bridge.');
console.log('PASS: auth headers are omitted from IPC arguments; no real credential enters the JavaScript bridge.');
console.log('LIMIT: The native IPC command is mocked; real ChatGPT OAuth, Keychain access, and Linux packaging remain unverified.');
