<?php

declare(strict_types=1);

/**
 * Tests for the HasStreamParsing trait.
 *
 * Exercises SSE (Server-Sent Events) and JSON object stream parsing
 * including chunked reads, edge cases, and error handling.
 */

use Core\Mod\Agentic\Services\Concerns\HasStreamParsing;
use Psr\Http\Message\StreamInterface;

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

/**
 * Create a minimal in-memory PSR-7 stream from a string.
 *
 * Only eof() and read() are needed by the trait; all other
 * StreamInterface methods are stubbed.
 */
function fakeStream(string $data, int $chunkSize = 8192): StreamInterface
{
    return new class($data, $chunkSize) implements StreamInterface
    {
        private int $pos = 0;

        public function __construct(
            private readonly string $data,
            private readonly int $chunkSize,
        ) {}

        public function eof(): bool
        {
            return $this->pos >= strlen($this->data);
        }

        public function read($length): string
        {
            $effective = min($length, $this->chunkSize);
            $chunk = substr($this->data, $this->pos, $effective);
            $this->pos += strlen($chunk);

            return $chunk;
        }

        // --- PSR-7 stubs (not exercised by the trait) ---
        public function __toString(): string
        {
            return $this->data;
        }

        public function close(): void {}

        public function detach()
        {
            return null;
        }

        public function getSize(): ?int
        {
            return strlen($this->data);
        }

        public function tell(): int
        {
            return $this->pos;
        }

        public function isSeekable(): bool
        {
            return false;
        }

        public function seek($offset, $whence = SEEK_SET): void {}

        public function rewind(): void {}

        public function isWritable(): bool
        {
            return false;
        }

        public function write($string): int
        {
            return 0;
        }

        public function isReadable(): bool
        {
            return true;
        }

        public function getContents(): string
        {
            return substr($this->data, $this->pos);
        }

        public function getMetadata($key = null)
        {
            return null;
        }
    };
}

/**
 * Create a testable object that exposes the HasStreamParsing trait methods.
 */
function streamParsingService(): object
{
    return new class
    {
        use HasStreamParsing;

        public function sse(StreamInterface $stream, callable $extract): Generator
        {
            return $this->parseSSEStream($stream, $extract);
        }

        public function json(StreamInterface $stream, callable $extract): Generator
        {
            return $this->parseJSONStream($stream, $extract);
        }
    };
}

// ---------------------------------------------------------------------------
// parseSSEStream – basic data extraction
// ---------------------------------------------------------------------------

describe('parseSSEStream basic parsing', function () {
    it('yields content from a single data line', function () {
        $raw = "data: {\"text\":\"hello\"}\n\n";
        $service = streamParsingService();

        $results = iterator_to_array(
            $service->sse(fakeStream($raw), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBe(['hello']);
    });

    it('yields content from multiple data lines', function () {
        $raw = "data: {\"text\":\"foo\"}\n";
        $raw .= "data: {\"text\":\"bar\"}\n";
        $raw .= "data: {\"text\":\"baz\"}\n";
        $service = streamParsingService();

        $results = iterator_to_array(
            $service->sse(fakeStream($raw), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBe(['foo', 'bar', 'baz']);
    });

    it('handles Windows-style \\r\\n line endings', function () {
        $raw = "data: {\"text\":\"crlf\"}\r\n\r\n";
        $service = streamParsingService();

        $results = iterator_to_array(
            $service->sse(fakeStream($raw), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBe(['crlf']);
    });
});

// ---------------------------------------------------------------------------
// parseSSEStream – stream termination
// ---------------------------------------------------------------------------

describe('parseSSEStream stream termination', function () {
    it('stops yielding when it encounters [DONE]', function () {
        $raw = "data: {\"text\":\"before\"}\n";
        $raw .= "data: [DONE]\n";
        $raw .= "data: {\"text\":\"after\"}\n";
        $service = streamParsingService();

        $results = iterator_to_array(
            $service->sse(fakeStream($raw), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBe(['before']);
    });

    it('stops when [DONE] has surrounding whitespace', function () {
        $raw = "data: {\"text\":\"first\"}\n";
        $raw .= "data:  [DONE] \n";
        $service = streamParsingService();

        $results = iterator_to_array(
            $service->sse(fakeStream($raw), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBe(['first']);
    });

    it('yields nothing from an empty stream', function () {
        $service = streamParsingService();

        $results = iterator_to_array(
            $service->sse(fakeStream(''), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBeEmpty();
    });
});

// ---------------------------------------------------------------------------
// parseSSEStream – skipped lines
// ---------------------------------------------------------------------------

describe('parseSSEStream skipped lines', function () {
    it('skips blank/separator lines', function () {
        $raw = "\n\ndata: {\"text\":\"ok\"}\n\n\n";
        $service = streamParsingService();

        $results = iterator_to_array(
            $service->sse(fakeStream($raw), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBe(['ok']);
    });

    it('skips non-data SSE fields (event:, id:, retry:)', function () {
        $raw = "event: message\n";
        $raw .= "id: 42\n";
        $raw .= "retry: 3000\n";
        $raw .= "data: {\"text\":\"content\"}\n";
        $service = streamParsingService();

        $results = iterator_to_array(
            $service->sse(fakeStream($raw), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBe(['content']);
    });

    it('skips SSE comment lines starting with colon', function () {
        $raw = ": keep-alive\n";
        $raw .= "data: {\"text\":\"real\"}\n";
        $service = streamParsingService();

        $results = iterator_to_array(
            $service->sse(fakeStream($raw), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBe(['real']);
    });

    it('skips data lines with empty payload after trimming', function () {
        $raw = "data:   \n";
        $raw .= "data: {\"text\":\"actual\"}\n";
        $service = streamParsingService();

        $results = iterator_to_array(
            $service->sse(fakeStream($raw), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBe(['actual']);
    });
});

// ---------------------------------------------------------------------------
// parseSSEStream – error handling
// ---------------------------------------------------------------------------

describe('parseSSEStream error handling', function () {
    it('skips lines with invalid JSON', function () {
        $raw = "data: not-valid-json\n";
        $raw .= "data: {\"text\":\"valid\"}\n";
        $service = streamParsingService();

        $results = iterator_to_array(
            $service->sse(fakeStream($raw), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBe(['valid']);
    });

    it('skips lines where extractor returns null', function () {
        $raw = "data: {\"other\":\"field\"}\n";
        $raw .= "data: {\"text\":\"present\"}\n";
        $service = streamParsingService();

        $results = iterator_to_array(
            $service->sse(fakeStream($raw), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBe(['present']);
    });

    it('skips lines where extractor returns empty string', function () {
        $raw = "data: {\"text\":\"\"}\n";
        $raw .= "data: {\"text\":\"hello\"}\n";
        $service = streamParsingService();

        $results = iterator_to_array(
            $service->sse(fakeStream($raw), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBe(['hello']);
    });
});

// ---------------------------------------------------------------------------
// parseSSEStream – chunked / partial reads
// ---------------------------------------------------------------------------

describe('parseSSEStream chunked reads', function () {
    it('handles a stream delivered in small chunks', function () {
        $raw = "data: {\"text\":\"chunked\"}\n\n";
        $service = streamParsingService();

        // Force the stream to return 5 bytes at a time
        $results = iterator_to_array(
            $service->sse(fakeStream($raw, 5), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBe(['chunked']);
    });

    it('processes remaining data buffered after stream EOF', function () {
        // No trailing newline – data stays in the buffer until EOF
        $raw = 'data: {"text":"buffered"}';
        $service = streamParsingService();

        $results = iterator_to_array(
            $service->sse(fakeStream($raw), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBe(['buffered']);
    });
});

// ---------------------------------------------------------------------------
// parseJSONStream – basic parsing
// ---------------------------------------------------------------------------

describe('parseJSONStream basic parsing', function () {
    it('yields content from a single JSON object', function () {
        $raw = '{"text":"hello"}';
        $service = streamParsingService();

        $results = iterator_to_array(
            $service->json(fakeStream($raw), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBe(['hello']);
    });

    it('yields content from multiple consecutive JSON objects', function () {
        $raw = '{"text":"first"}{"text":"second"}{"text":"third"}';
        $service = streamParsingService();

        $results = iterator_to_array(
            $service->json(fakeStream($raw), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBe(['first', 'second', 'third']);
    });

    it('handles JSON objects separated by whitespace', function () {
        $raw = "  {\"text\":\"a\"}\n\n  {\"text\":\"b\"}\n";
        $service = streamParsingService();

        $results = iterator_to_array(
            $service->json(fakeStream($raw), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBe(['a', 'b']);
    });

    it('handles nested JSON objects correctly', function () {
        $raw = '{"outer":{"inner":"value"},"text":"top"}';
        $service = streamParsingService();

        $results = iterator_to_array(
            $service->json(fakeStream($raw), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBe(['top']);
    });

    it('handles escaped quotes inside strings', function () {
        $raw = '{"text":"say \"hello\""}';
        $service = streamParsingService();

        $results = iterator_to_array(
            $service->json(fakeStream($raw), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBe(['say "hello"']);
    });
});

// ---------------------------------------------------------------------------
// parseJSONStream – extractor filtering
// ---------------------------------------------------------------------------

describe('parseJSONStream extractor filtering', function () {
    it('skips objects where extractor returns null', function () {
        $raw = '{"other":"x"}{"text":"keep"}';
        $service = streamParsingService();

        $results = iterator_to_array(
            $service->json(fakeStream($raw), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBe(['keep']);
    });

    it('skips objects where extractor returns empty string', function () {
        $raw = '{"text":""}{"text":"non-empty"}';
        $service = streamParsingService();

        $results = iterator_to_array(
            $service->json(fakeStream($raw), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBe(['non-empty']);
    });

    it('yields nothing from an empty stream', function () {
        $service = streamParsingService();

        $results = iterator_to_array(
            $service->json(fakeStream(''), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBeEmpty();
    });
});

// ---------------------------------------------------------------------------
// parseJSONStream – chunked reads
// ---------------------------------------------------------------------------

describe('parseJSONStream chunked reads', function () {
    it('handles objects split across multiple chunks', function () {
        $raw = '{"text":"split"}';
        $service = streamParsingService();

        // Force 3-byte chunks to ensure the object is assembled across reads
        $results = iterator_to_array(
            $service->json(fakeStream($raw, 3), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBe(['split']);
    });

    it('handles multiple objects across chunks', function () {
        $raw = '{"text":"a"}{"text":"b"}';
        $service = streamParsingService();

        $results = iterator_to_array(
            $service->json(fakeStream($raw, 4), fn ($json) => $json['text'] ?? null)
        );

        expect($results)->toBe(['a', 'b']);
    });
});
