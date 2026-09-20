import { useState, useEffect } from 'react';
import { Solution } from '../types/api';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '';
const SOLUTIONS_ENDPOINT = `${API_BASE_URL}/solutions`;

export function useSolutions(board?: string[][]) {
  const [solutions, setSolutions] = useState<Solution[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    const controller = new AbortController();

    const fetchSolutions = async () => {
      setIsLoading(true);
      setError(null);
      setSolutions([]);

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
        if (!controller.signal.aborted) {
          setIsLoading(false);
        }
      }
    };

    fetchSolutions();

    return () => {
      controller.abort();
    };
  }, [board]); // Add board as dependency

  return {
    data: solutions,
    isLoading,
    error,
  };
}
