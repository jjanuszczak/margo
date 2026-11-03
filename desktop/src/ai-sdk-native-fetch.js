const RESPONSES_URL = 'https://api.openai.com/v1/responses';

// The native command owns credentials and upstream HTTP. Only the request URL,
// body, and auth mode cross Tauri IPC; the SDK's placeholder Authorization
// header is deliberately discarded here.
export function createNativeOpenAIFetch({ invoke, listen, authMode, sessionOnly }) {
  return async (input, init={}) => {
    const request = new Request(input, init);
    if (request.method !== 'POST' || new URL(request.url).href !== RESPONSES_URL) {
      throw new Error('The native ChatGPT bridge only accepts OpenAI Responses requests.');
    }

    const requestId = crypto.randomUUID();
    const eventName = `margo-ai-sdk-${requestId}`;
    let controller;
    let unlisten;
    let completed = false;
    let resolveHeaders;
    let rejectHeaders;
    const headersReady = new Promise((resolve, reject) => {
      resolveHeaders = resolve;
      rejectHeaders = reject;
    });
    const body = new ReadableStream({
      start(streamController) { controller = streamController; },
      cancel() {
        completed = true;
        unlisten?.();
      },
    });

    try {
      unlisten = await listen(eventName, event => {
        if (completed) return;
        const payload = event.payload || {};
        if (payload.kind === 'headers') {
          resolveHeaders({ status: payload.status, headers: payload.headers || {} });
        } else if (payload.kind === 'chunk') {
          const binary = atob(payload.base64 || '');
          controller.enqueue(Uint8Array.from(binary, character => character.charCodeAt(0)));
        } else if (payload.kind === 'done') {
          completed = true;
          controller.close();
          unlisten?.();
        } else if (payload.kind === 'error') {
          const error = new Error(payload.message || 'Native provider request failed.');
          completed = true;
          rejectHeaders(error);
          controller.error(error);
          unlisten?.();
        }
      });

      // Never forward request.headers: it contains the SDK's placeholder key.
      await invoke('ai_sdk_proxy_start', {
        requestId,
        url: request.url,
        body: await request.text(),
        authMode,
        sessionOnly,
      });
      const metadata = await headersReady;
      return new Response(body, {
        status: metadata.status,
        headers: metadata.headers,
      });
    } catch (error) {
      completed = true;
      controller.error(error);
      unlisten?.();
      throw error;
    }
  };
}
