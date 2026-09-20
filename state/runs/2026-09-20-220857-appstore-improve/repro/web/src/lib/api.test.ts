import { describe, expect, it, vi } from "vitest";
import { api, streamAiChat } from "./api";

describe("API client", () => {
  it("translates UI page state to the backend limit/offset page contract", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          items: [
            {
              id: "1",
              slug: "agent-hub",
              name: "Agent Hub",
              summary: "AI catalog",
            },
          ],
          total: 55,
          limit: 24,
          offset: 24,
        }),
        { status: 200, headers: { "content-type": "application/json" } },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    const page = await api.apps({ q: "agent", page: 2, pageSize: 24 });

    expect(fetchMock).toHaveBeenCalledOnce();
    expect(String(fetchMock.mock.calls[0]?.[0])).toContain("q=agent");
    expect(String(fetchMock.mock.calls[0]?.[0])).toContain("limit=24");
    expect(String(fetchMock.mock.calls[0]?.[0])).toContain("offset=24");
    expect(page).toMatchObject({ total: 55, page: 2, pageSize: 24 });
  });

  it("parses token, usage and finish SSE frames", async () => {
    const encoder = new TextEncoder();
    const body = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(
          encoder.encode('event: token\ndata: {"text":"안녕"}\n\n'),
        );
        controller.enqueue(
          encoder.encode('event: usage\ndata: {"totalTokens":3}\n\n'),
        );
        controller.enqueue(encoder.encode("data: [DONE]\n\n"));
        controller.close();
      },
    });
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(body, {
          status: 200,
          headers: { "content-type": "text/event-stream" },
        }),
      ),
    );
    const events: Array<{ event: string; data: unknown }> = [];

    await streamAiChat(
      { messages: [{ role: "user", content: "hello" }] },
      (event) => events.push(event),
      new AbortController().signal,
    );

    expect(events).toEqual([
      { event: "token", data: { text: "안녕" } },
      { event: "usage", data: { totalTokens: 3 } },
      { event: "finish", data: null },
    ]);
  });

  async function collectStream(chunks: Array<string | Uint8Array>) {
    const encoder = new TextEncoder();
    const body = new ReadableStream<Uint8Array>({
      start(controller) {
        for (const chunk of chunks)
          controller.enqueue(
            typeof chunk === "string" ? encoder.encode(chunk) : chunk,
          );
        controller.close();
      },
    });
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(body, {
          status: 200,
          headers: { "content-type": "text/event-stream" },
        }),
      ),
    );
    const events: Array<{ event: string; data: unknown }> = [];
    await streamAiChat(
      { messages: [{ role: "user", content: "hello" }] },
      (event) => events.push(event),
      new AbortController().signal,
    );
    return events;
  }

  it("delivers the last event when the stream closes without a trailing blank line", async () => {
    const events = await collectStream([
      'event: token\ndata: {"text":"안"}\n\n',
      // The connection closes right after this event: no "\n\n" follows.
      'event: token\ndata: {"text":"녕"}',
    ]);

    expect(events).toEqual([
      { event: "token", data: { text: "안" } },
      { event: "token", data: { text: "녕" } },
    ]);
  });

  it("emits finish when the stream ends on an unterminated [DONE]", async () => {
    const events = await collectStream([
      'event: token\ndata: {"text":"안녕"}\n\n',
      "data: [DO",
      "NE]",
    ]);

    expect(events).toEqual([
      { event: "token", data: { text: "안녕" } },
      { event: "finish", data: null },
    ]);
  });

  it("drops a trailing block whose JSON was cut off by the connection closing", async () => {
    const events = await collectStream([
      'event: token\ndata: {"text":"안녕"}\n\n',
      // The connection dropped mid-event: the fragment must not be shown as text.
      'event: token\ndata: {"type":"token","text":"안',
    ]);

    expect(events).toEqual([{ event: "token", data: { text: "안녕" } }]);
  });

  it("joins an event split across chunks in the middle of a multibyte character", async () => {
    const encoded = new TextEncoder().encode(
      'event: token\ndata: {"text":"안녕"}\n\n',
    );
    // Cut inside the 3-byte UTF-8 sequence for "안" so the decoder must buffer.
    const cut = 'event: token\ndata: {"text":"'.length + 1;
    const events = await collectStream([
      encoded.slice(0, cut),
      encoded.slice(cut),
    ]);

    expect(events).toEqual([{ event: "token", data: { text: "안녕" } }]);
  });
});
