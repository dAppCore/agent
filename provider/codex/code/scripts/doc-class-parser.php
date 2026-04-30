<?php

if ($argc < 2) {
    echo "Usage: php doc-class-parser.php <file_path>\n";
    exit(1);
}

$filePath = $argv[1];
if (!file_exists($filePath)) {
    echo "Error: File not found at '$filePath'\n";
    exit(1);
}

// --- Find the namespace and class name by parsing the file ---
$fileContent = file_get_contents($filePath);

$namespace = '';
if (preg_match('/^\s*namespace\s+([^;]+);/m', $fileContent, $namespaceMatches)) {
    $namespace = $namespaceMatches[1];
}

$className = '';
if (!preg_match('/class\s+([a-zA-Z_\x7f-\xff][a-zA-Z0-9_\x7f-\xff]*)/', $fileContent, $matches)) {
    echo "Error: Could not find class name in '$filePath'\n";
    exit(1);
}
$className = $matches[1];

$fqcn = $namespace ? $namespace . '\\' . $className : $className;

// Now that we have the class name, we can require the file.
require_once $filePath;

// --- Utility function to parse docblocks ---
function parseDocComment($docComment) {
    $data = [
        'description' => '',
        'params' => [],
        'return' => null,
    ];
    if (!$docComment) return $data;

    $lines = array_map(function($line) {
        return trim(substr(trim($line), 1));
    }, explode("\n", $docComment));

    $descriptionDone = false;
    foreach ($lines as $line) {
        if ($line === '/**' || $line === '*/' || $line === '*') continue;

        if (strpos($line, '@') === 0) {
            $descriptionDone = true;
            preg_match('/@(\w+)\s*(.*)/', $line, $matches);
            if (count($matches) === 3) {
                $tag = $matches[1];
                $content = trim($matches[2]);

                if ($tag === 'param') {
                    preg_match('/(\S+)\s+\$(\S+)\s*(.*)/', $content, $paramMatches);
                    if(count($paramMatches) >= 3) {
                       $data['params'][$paramMatches[2]] = [
                           'type' => $paramMatches[1],
                           'description' => $paramMatches[3] ?? ''
                       ];
                    }
                } elseif ($tag === 'return') {
                    preg_match('/(\S+)\s*(.*)/', $content, $returnMatches);
                    if(count($returnMatches) >= 2) {
                        $data['return'] = [
                            'type' => $returnMatches[1],
                            'description' => $returnMatches[2] ?? ''
                        ];
                    }
                }
            }
        } elseif (!$descriptionDone) {
            $data['description'] .= $line . " ";
        }
    }
    $data['description'] = trim($data['description']);
    return $data;
}

// --- Use Reflection API to get class details ---
try {
    if (!class_exists($fqcn)) {
        echo "Error: Class '$fqcn' does not exist after including file '$filePath'.\n";
        exit(1);
    }
    $reflectionClass = new ReflectionClass($fqcn);
} catch (ReflectionException $e) {
    echo "Error: " . $e->getMessage() . "\n";
    exit(1);
}

$classDocData = parseDocComment($reflectionClass->getDocComment());

$methodsData = [];
$publicMethods = $reflectionClass->getMethods(ReflectionMethod::IS_PUBLIC);

foreach ($publicMethods as $method) {
    $methodDocData = parseDocComment($method->getDocComment());
    $paramsData = [];

    foreach ($method->getParameters() as $param) {
        $paramName = $param->getName();
        $paramInfo = [
            'type' => ($param->getType() ? (string)$param->getType() : ($methodDocData['params'][$paramName]['type'] ?? 'mixed')),
            'required' => !$param->isOptional(),
            'description' => $methodDocData['params'][$paramName]['description'] ?? ''
        ];
        $paramsData[$paramName] = $paramInfo;
    }

    $methodsData[] = [
        'name' => $method->getName(),
        'description' => $methodDocData['description'],
        'params' => $paramsData,
        'return' => $methodDocData['return']
    ];
}

// --- Output as JSON ---
$output = [
    'className' => $reflectionClass->getShortName(),
    'description' => $classDocData['description'],
    'methods' => $methodsData,
];

echo json_encode($output, JSON_PRETTY_PRINT);
