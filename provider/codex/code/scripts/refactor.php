#!/usr/bin/env php
<?php

require_once __DIR__ . '/../../../vendor/autoload.php';

use PhpParser\ParserFactory;
use PhpParser\Node;
use PhpParser\Node\Stmt\Class_;
use PhpParser\Node\Stmt\ClassMethod;
use PhpParser\PrettyPrinter;
use PhpParser\NodeVisitorAbstract;

class MethodExtractor extends NodeVisitorAbstract
{
    private $startLine;
    private $endLine;
    private $newMethodName;

    public function __construct($startLine, $endLine, $newMethodName)
    {
        $this->startLine = $startLine;
        $this->endLine = $endLine;
        $this->newMethodName = $newMethodName;
    }

    public function leaveNode(Node $node)
    {
        if ($node instanceof Class_) {
            $classNode = $node;
            $originalMethod = null;
            $extractionStartIndex = -1;
            $extractionEndIndex = -1;

            foreach ($classNode->stmts as $stmt) {
                if ($stmt instanceof ClassMethod) {
                    foreach ($stmt->stmts as $index => $mstmt) {
                        if ($mstmt->getStartLine() >= $this->startLine && $extractionStartIndex === -1) {
                            $extractionStartIndex = $index;
                        }
                        if ($mstmt->getEndLine() <= $this->endLine && $extractionStartIndex !== -1) {
                            $extractionEndIndex = $index;
                        }
                    }

                    if ($extractionStartIndex !== -1) {
                        $originalMethod = $stmt;
                        break;
                    }
                }
            }

            if ($originalMethod !== null) {
                $statementsToExtract = array_slice(
                    $originalMethod->stmts,
                    $extractionStartIndex,
                    $extractionEndIndex - $extractionStartIndex + 1
                );

                $newMethod = new ClassMethod($this->newMethodName, [
                    'stmts' => $statementsToExtract
                ]);
                $classNode->stmts[] = $newMethod;

                $methodCall = new Node\Expr\MethodCall(new Node\Expr\Variable('this'), $this->newMethodName);
                $methodCallStatement = new Node\Stmt\Expression($methodCall);

                array_splice(
                    $originalMethod->stmts,
                    $extractionStartIndex,
                    count($statementsToExtract),
                    [$methodCallStatement]
                );
            }
        }
    }
}


$subcommand = $argv[1] ?? null;

switch ($subcommand) {
    case 'extract-method':
        $filePath = 'Test.php';
        $startLine = 9;
        $endLine = 13;
        $newMethodName = 'newMethod';

        $code = file_get_contents($filePath);

        $parser = (new ParserFactory)->create(ParserFactory::PREFER_PHP7);
        $ast = $parser->parse($code);

        $traverser = new PhpParser\NodeTraverser();
        $traverser->addVisitor(new MethodExtractor($startLine, $endLine, $newMethodName));

        $modifiedAst = $traverser->traverse($ast);

        $prettyPrinter = new PrettyPrinter\Standard;
        $newCode = $prettyPrinter->prettyPrintFile($modifiedAst);

        file_put_contents($filePath, $newCode);

        echo "Refactoring complete.\n";
        break;
    default:
        echo "Unknown subcommand: $subcommand\n";
        exit(1);
}
