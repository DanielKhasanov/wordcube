import { describe, it, expect, vi, afterEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useSolutions } from './useSolutions';
import { Solution } from '../types/api';

function ndjsonStream(lines: string[]): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder();
  return new ReadableStream({
    start(controller) {
      for (const line of lines) {
        controller.enqueue(encoder.encode(line + '\n'));
      }
      controller.close();
    },
  });
}

function jsonLine(solution: Solution): string {
  return JSON.stringify(solution);
}

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe('useSolutions', () => {
  it('parses a streamed NDJSON response into solutions incrementally', async () => {
    const solutions: Solution[] = [
      { id: 1, grid: [['a']] },
      { id: 2, grid: [['b']] },
    ];
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(ndjsonStream(solutions.map(jsonLine)), {
        status: 200,
        headers: { 'Content-Type': 'application/x-ndjson' },
      })
    );
    vi.stubGlobal('fetch', fetchMock);

    const { result } = renderHook(() => useSolutions());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.error).toBeNull();
    expect(result.current.data).toEqual(solutions);
  });

  it('ignores malformed lines in the stream without surfacing an error', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(ndjsonStream(['not json', jsonLine({ id: 1, grid: [['a']] })]), {
        status: 200,
      })
    );
    vi.stubGlobal('fetch', fetchMock);

    const { result } = renderHook(() => useSolutions());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.error).toBeNull();
    expect(result.current.data).toEqual([{ id: 1, grid: [['a']] }]);
  });

  it('surfaces an error when the server responds with a non-ok status', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 503 }));
    vi.stubGlobal('fetch', fetchMock);

    const { result } = renderHook(() => useSolutions());

    await waitFor(() => expect(result.current.isLoading).toBe(false));

    expect(result.current.error).not.toBeNull();
    expect(result.current.data).toEqual([]);
  });

  it('aborts the in-flight request when the board changes before it completes', async () => {
    let capturedSignal: AbortSignal | undefined;
    const fetchMock = vi.fn().mockImplementation((_url, init?: RequestInit) => {
      capturedSignal = init?.signal ?? undefined;
      return new Promise(() => {
        // Never resolves: simulates a slow/streaming request that gets superseded.
      });
    });
    vi.stubGlobal('fetch', fetchMock);

    const board1 = Array(5).fill(null).map(() => Array(5).fill(''));
    const { rerender, unmount } = renderHook(({ board }) => useSolutions(board), {
      initialProps: { board: board1 },
    });

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    expect(capturedSignal?.aborted).toBe(false);
    const firstSignal = capturedSignal;

    const board2 = Array(5).fill(null).map(() => Array(5).fill(''));
    board2[0][0] = 'a';
    rerender({ board: board2 });

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
    expect(firstSignal?.aborted).toBe(true);

    unmount();
  });

  it('does not surface an error state when a request is aborted', async () => {
    const fetchMock = vi.fn().mockImplementation((_url, init?: RequestInit) => {
      return new Promise((_resolve, reject) => {
        init?.signal?.addEventListener('abort', () => {
          reject(new DOMException('Aborted', 'AbortError'));
        });
      });
    });
    vi.stubGlobal('fetch', fetchMock);

    const board1 = Array(5).fill(null).map(() => Array(5).fill(''));
    const { rerender, result } = renderHook(({ board }) => useSolutions(board), {
      initialProps: { board: board1 },
    });

    const board2 = Array(5).fill(null).map(() => Array(5).fill(''));
    board2[0][0] = 'a';
    rerender({ board: board2 });

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
    await new Promise((r) => setTimeout(r, 0));

    expect(result.current.error).toBeNull();
  });
});
