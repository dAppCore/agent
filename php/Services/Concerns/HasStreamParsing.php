<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Services\Concerns;

use Generator;
use Psr\Http\Message\StreamInterface;

/**
 * Provides robust SSE (Server-Sent Events) stream parsing for AI provider services.
 *
 * Handles:
 * - Chunked/partial reads
 * - Line buffering across chunks
 * - SSE event parsing (data:, event:, etc.)
 */
trait HasStreamParsing
{
    /**
     * Parse SSE stream and yield data payloads.
     *
     * @param  StreamInterface  $stream  The HTTP response body stream
     * @param  callable  $extractContent  Function to extract content from parsed JSON data
     * @return Generator<string>
     */
    protected function parseSSEStream(StreamInterface $stream, callable $extractContent): Generator
    {
        $buffer = '';

        while (! $stream->eof()) {
            $chunk = $stream->read(8192);

            if ($chunk === '') {
                continue;
            }

            $buffer .= $chunk;

            // Process complete lines from the buffer
            while (($newlinePos = strpos($buffer, "\n")) !== false) {
                $line = substr($buffer, 0, $newlinePos);
                $buffer = substr($buffer, $newlinePos + 1);

                // Trim carriage return if present (handle \r\n)
                $line = rtrim($line, "\r");

                // Skip empty lines (event separators)
                if ($line === '') {
                    continue;
                }

                // Parse SSE data lines
                if (str_starts_with($line, 'data: ')) {
                    $data = substr($line, 6);

                    // Check for stream termination
                    if ($data === '[DONE]' || trim($data) === '[DONE]') {
                        return;
                    }

                    // Skip empty data
                    if (trim($data) === '') {
                        continue;
                    }

                    // Parse JSON payload
                    $json = json_decode($data, true);

                    if ($json === null && json_last_error() !== JSON_ERROR_NONE) {
                        // Invalid JSON, skip this line
                        continue;
                    }

                    // Extract content using provider-specific callback
                    $content = $extractContent($json);

                    if ($content !== null && $content !== '') {
                        yield $content;
                    }
                }

                // Skip other SSE fields (event:, id:, retry:, comments starting with :)
            }
        }

        // Process any remaining data in buffer after stream ends
        if (trim($buffer) !== '') {
            $lines = explode("\n", $buffer);
            foreach ($lines as $line) {
                $line = rtrim($line, "\r");
                if (str_starts_with($line, 'data: ')) {
                    $data = substr($line, 6);
                    if ($data !== '[DONE]' && trim($data) !== '' && trim($data) !== '[DONE]') {
                        $json = json_decode($data, true);
                        if ($json !== null) {
                            $content = $extractContent($json);
                            if ($content !== null && $content !== '') {
                                yield $content;
                            }
                        }
                    }
                }
            }
        }
    }

    /**
     * Parse JSON object stream (for providers like Gemini that don't use SSE).
     *
     * @param  StreamInterface  $stream  The HTTP response body stream
     * @param  callable  $extractContent  Function to extract content from parsed JSON data
     * @return Generator<string>
     */
    protected function parseJSONStream(StreamInterface $stream, callable $extractContent): Generator
    {
        $buffer = '';
        $braceDepth = 0;
        $inString = false;
        $escape = false;
        $objectStart = -1;
        $scanPos = 0;

        while (! $stream->eof()) {
            $chunk = $stream->read(8192);

            if ($chunk === '') {
                continue;
            }

            $buffer .= $chunk;

            // Parse JSON objects from the buffer, continuing from where
            // the previous iteration left off to preserve parser state.
            $length = strlen($buffer);
            $i = $scanPos;

            while ($i < $length) {
                $char = $buffer[$i];

                if ($escape) {
                    $escape = false;
                    $i++;

                    continue;
                }

                if ($char === '\\' && $inString) {
                    $escape = true;
                    $i++;

                    continue;
                }

                if ($char === '"') {
                    $inString = ! $inString;
                } elseif (! $inString) {
                    if ($char === '{') {
                        if ($braceDepth === 0) {
                            $objectStart = $i;
                        }
                        $braceDepth++;
                    } elseif ($char === '}') {
                        $braceDepth--;
                        if ($braceDepth === 0 && $objectStart >= 0) {
                            // Complete JSON object found
                            $jsonStr = substr($buffer, $objectStart, $i - $objectStart + 1);
                            $json = json_decode($jsonStr, true);

                            if ($json !== null) {
                                $content = $extractContent($json);
                                if ($content !== null && $content !== '') {
                                    yield $content;
                                }
                            }

                            // Update buffer to remove processed content
                            $buffer = substr($buffer, $i + 1);
                            $length = strlen($buffer);
                            $i = -1; // Will be incremented to 0
                            $scanPos = 0;
                            $objectStart = -1;
                        }
                    }
                }

                $i++;
            }

            // Save scan position so we resume from here on the next chunk
            $scanPos = $i;
        }
    }
}
