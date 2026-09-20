import { useState, useEffect } from 'react';
import { Solution } from '../types/api';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '';
const SOLUTIONS_ENDPOINT = `${API_BASE_URL}/solutions`;

// The backend loads and indexes its multi-million-row solution set in a
// background goroutine at startup (see GET /status) and returns 503 from
// /solutions until that finishes - a normal, expected window (observed to
// last several seconds against the real solutions.textpb), not a failure.
// Retry after a short delay instead of surfacing it as an error.
const NOT_READY_RETRY_DELAY_MS = 1000;

export function useSolutions(board?: string[][]) {
  const [solutions, setSolutions] = useState<Solution[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    let retryTimeout: ReturnType<typeof setTimeout> | undefined;

    const fetchSolutions = async () => {
      let retrying = false;
      try {
        const requestBody = {
          board: board || Array(5).fill(null).map(() => Array(5).fill(''))
        };

        const response = await fetch(SOLUTIONS_ENDPOINT, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(requestBody),
          signal: controller.signal,
        });

        if (response.status === 503) {
          retrying = true;
          retryTimeout = setTimeout(fetchSolutions, NOT_READY_RETRY_DELAY_MS);
          return;
        }

        if (!response.ok) {
          throw new Error('Failed to fetch solutions');
        }

        const reader = response.body?.getReader();
        if (!reader) {
          throw new Error('No response body');
        }

        const decoder = new TextDecoder();
        let buffer = '';

        try {
          while (true) {
            const { done, value } = await reader.read();

            if (done) {
              break;
            }

            buffer += decoder.decode(value, { stream: true });

            // Split by newlines and process each complete JSON object
            const lines = buffer.split('\n');
            buffer = lines.pop() || ''; // Keep the last incomplete line

            for (const line of lines) {
              if (line.trim()) {
                try {
                  const solution = JSON.parse(line.trim()) as Solution;
                  // Add solution immediately to trigger re-render
                  setSolutions(prev => [...prev, solution]);
                } catch (e) {
                  console.warn('Failed to parse solution:', line, e);
                }
              }
            }
          }
        } finally {
          reader.releaseLock();
        }
      } catch (err) {
        if (err instanceof DOMException && err.name === 'AbortError') {
          return;
        }
        setError(err instanceof Error ? err : new Error('Unknown error'));
      } finally {
        if (!retrying && !controller.signal.aborted) {
          setIsLoading(false);
        }
      }
    };

    setIsLoading(true);
    setError(null);
    setSolutions([]);
    fetchSolutions();

    return () => {
      controller.abort();
      clearTimeout(retryTimeout);
    };
  }, [board]); // Add board as dependency

  return {
    data: solutions,
    isLoading,
    error,
  };
}
