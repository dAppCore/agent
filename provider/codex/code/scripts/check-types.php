<?php

if ($argc < 2) {
    echo "Usage: php " . $argv[0] . " <file_path> [--auto-fix]\n";
    exit(1);
}

$filePath = $argv[1];
$autoFix = isset($argv[2]) && $argv[2] === '--auto-fix';

if (!file_exists($filePath)) {
    echo "Error: File not found at " . $filePath . "\n";
    exit(1);
}

$content = file_get_contents($filePath);
$tokens = token_get_all($content);

function checkStrictTypes(array $tokens, string $filePath, bool $autoFix, string &$content): void
{
    $hasStrictTypes = false;
    foreach ($tokens as $i => $token) {
        if (!is_array($token) || $token[0] !== T_DECLARE) {
            continue;
        }

        // Found a declare statement, now check if it's strict_types=1
        $next = findNextMeaningfulToken($tokens, $i + 1);
        if ($next && is_string($tokens[$next]) && $tokens[$next] === '(') {
            $next = findNextMeaningfulToken($tokens, $next + 1);
            if ($next && is_array($tokens[$next]) && $tokens[$next][0] === T_STRING && $tokens[$next][1] === 'strict_types') {
                $next = findNextMeaningfulToken($tokens, $next + 1);
                if ($next && is_string($tokens[$next]) && $tokens[$next] === '=') {
                    $next = findNextMeaningfulToken($tokens, $next + 1);
                    if ($next && is_array($tokens[$next]) && $tokens[$next][0] === T_LNUMBER && $tokens[$next][1] === '1') {
                        $hasStrictTypes = true;
                        break;
                    }
                }
            }
        }
    }

    if (!$hasStrictTypes) {
        fwrite(STDERR, "⚠ Line 1: Missing declare(strict_types=1)\n");
        if ($autoFix) {
            $content = str_replace('<?php', "<?php\n\ndeclare(strict_types=1);", $content);
            file_put_contents($filePath, $content);
            fwrite(STDERR, "✓ Auto-fixed: Added declare(strict_types=1)\n");
        }
    }
}

function findNextMeaningfulToken(array $tokens, int $index): ?int
{
    for ($i = $index; $i < count($tokens); $i++) {
        if (is_array($tokens[$i]) && in_array($tokens[$i][0], [T_WHITESPACE, T_COMMENT, T_DOC_COMMENT])) {
            continue;
        }
        return $i;
    }
    return null;
}

function checkParameterTypeHints(array $tokens): void
{
    foreach ($tokens as $i => $token) {
        if (!is_array($token) || $token[0] !== T_FUNCTION) {
            continue;
        }

        $parenStart = findNextMeaningfulToken($tokens, $i + 1);
        if (!$parenStart || !is_array($tokens[$parenStart]) || $tokens[$parenStart][0] !== T_STRING) {
            continue; // Not a standard function definition, maybe an anonymous function
        }

        $parenStart = findNextMeaningfulToken($tokens, $parenStart + 1);
        if (!$parenStart || !is_string($tokens[$parenStart]) || $tokens[$parenStart] !== '(') {
            continue;
        }

        $paramIndex = $parenStart + 1;
        while (true) {
            $nextParam = findNextMeaningfulToken($tokens, $paramIndex);
            if (!$nextParam || (is_string($tokens[$nextParam]) && $tokens[$nextParam] === ')')) {
                break; // End of parameter list
            }

            // We are at the start of a parameter declaration. It could be a type hint or the variable itself.
            $currentToken = $tokens[$nextParam];
            if (is_array($currentToken) && $currentToken[0] === T_VARIABLE) {
                // This variable has no type hint.
                fwrite(STDERR, "⚠ Line {$currentToken[2]}: Parameter {$currentToken[1]} has no type hint\n");
            }

            // Move to the next parameter
            $comma = findNextToken($tokens, $nextParam, ',');
            $closingParen = findNextToken($tokens, $nextParam, ')');

            if ($comma !== null && $comma < $closingParen) {
                $paramIndex = $comma + 1;
            } else {
                break; // No more commas, so no more parameters
            }
        }
    }
}

function findNextToken(array $tokens, int $index, $tokenType): ?int
{
    for ($i = $index; $i < count($tokens); $i++) {
        if (is_string($tokens[$i]) && $tokens[$i] === $tokenType) {
            return $i;
        }
        if (is_array($tokens[$i]) && $tokens[$i][0] === $tokenType) {
            return $i;
        }
    }
    return null;
}

function checkReturnTypeHints(array $tokens, string $filePath, bool $autoFix, string &$content): void
{
    foreach ($tokens as $i => $token) {
        if (!is_array($token) || $token[0] !== T_FUNCTION) {
            continue;
        }

        $functionNameToken = findNextMeaningfulToken($tokens, $i + 1);
        if (!$functionNameToken || !is_array($tokens[$functionNameToken]) || $tokens[$functionNameToken][0] !== T_STRING) {
            continue; // Not a standard function definition
        }
        $functionName = $tokens[$functionNameToken][1];
        if (in_array($functionName, ['__construct', '__destruct'])) {
            continue; // Constructors and destructors do not have return types
        }

        $parenStart = findNextMeaningfulToken($tokens, $functionNameToken + 1);
        if (!$parenStart || !is_string($tokens[$parenStart]) || $tokens[$parenStart] !== '(') {
            continue;
        }

        $parenEnd = findNextToken($tokens, $parenStart + 1, ')');
        if ($parenEnd === null) {
            continue; // Malformed function
        }

        $nextToken = findNextMeaningfulToken($tokens, $parenEnd + 1);
        if (!$nextToken || !(is_string($tokens[$nextToken]) && $tokens[$nextToken] === ':')) {
            fwrite(STDERR, "⚠ Line {$tokens[$functionNameToken][2]}: Method {$functionName}() has no return type\n");
            if ($autoFix) {
                // Check if the function has a return statement
                $bodyStart = findNextToken($tokens, $parenEnd + 1, '{');
                if ($bodyStart !== null) {
                    $bodyEnd = findMatchingBrace($tokens, $bodyStart);
                    if ($bodyEnd !== null) {
                        $hasReturn = false;
                        for ($j = $bodyStart; $j < $bodyEnd; $j++) {
                            if (is_array($tokens[$j]) && $tokens[$j][0] === T_RETURN) {
                                $hasReturn = true;
                                break;
                            }
                        }
                        if (!$hasReturn) {
                            $offset = 0;
                            for ($k = 0; $k < $parenEnd; $k++) {
                                if (is_array($tokens[$k])) {
                                    $offset += strlen($tokens[$k][1]);
                                } else {
                                    $offset += strlen($tokens[$k]);
                                }
                            }

                            $original = ')';
                            $replacement = ') : void';
                            $content = substr_replace($content, $replacement, $offset, strlen($original));

                            file_put_contents($filePath, $content);
                            fwrite(STDERR, "✓ Auto-fixed: Added : void return type to {$functionName}()\n");
                        }
                    }
                }
            }
        }
    }
}

function findMatchingBrace(array $tokens, int $startIndex): ?int
{
    $braceLevel = 0;
    for ($i = $startIndex; $i < count($tokens); $i++) {
        if (is_string($tokens[$i]) && $tokens[$i] === '{') {
            $braceLevel++;
        } elseif (is_string($tokens[$i]) && $tokens[$i] === '}') {
            $braceLevel--;
            if ($braceLevel === 0) {
                return $i;
            }
        }
    }
    return null;
}

function checkPropertyTypeHints(array $tokens): void
{
    foreach ($tokens as $i => $token) {
        if (!is_array($token) || !in_array($token[0], [T_PUBLIC, T_PROTECTED, T_PRIVATE, T_VAR])) {
            continue;
        }

        $nextToken = findNextMeaningfulToken($tokens, $i + 1);
        if ($nextToken && is_array($tokens[$nextToken]) && $tokens[$nextToken][0] === T_STATIC) {
            $nextToken = findNextMeaningfulToken($tokens, $nextToken + 1);
        }

        if ($nextToken && is_array($tokens[$nextToken]) && $tokens[$nextToken][0] === T_VARIABLE) {
            // This is a property without a type hint
            fwrite(STDERR, "⚠ Line {$tokens[$nextToken][2]}: Property {$tokens[$nextToken][1]} has no type hint\n");
        }
    }
}

function tokensToCode(array $tokens): string
{
    $code = '';
    foreach ($tokens as $token) {
        if (is_array($token)) {
            $code .= $token[1];
        } else {
            $code .= $token;
        }
    }
    return $code;
}

checkStrictTypes($tokens, $filePath, $autoFix, $content);
checkParameterTypeHints($tokens);
checkReturnTypeHints($tokens, $filePath, $autoFix, $content);
checkPropertyTypeHints($tokens);
