<?php

// Find the project's vendor/autoload.php to bootstrap the application's classes
function find_autoload($dir) {
    if (file_exists($dir . '/vendor/autoload.php')) {
        return $dir . '/vendor/autoload.php';
    }
    if (realpath($dir) === '/') {
        return false;
    }
    return find_autoload(dirname($dir));
}

$autoload_path = find_autoload(getcwd());
if (!$autoload_path) {
    echo json_encode(['error' => 'Could not find vendor/autoload.php. Ensure script is run from within a Laravel project.']);
    exit(1);
}
require_once $autoload_path;

if ($argc < 2) {
    echo json_encode(['error' => 'Model file path is required.']);
    exit(1);
}

$modelPath = $argv[1];
if (!file_exists($modelPath)) {
    echo json_encode(['error' => "Model file not found at $modelPath"]);
    exit(1);
}

// Convert file path to a class name (e.g., app/Models/User.php -> App\Models\User)
$className = str_replace('.php', '', $modelPath);
$className = ucfirst($className);
$className = str_replace('/', '\\', $className);


if (!class_exists($className)) {
    echo json_encode(['error' => "Class '$className' could not be found. Check the path and namespace."]);
    exit(1);
}

try {
    $reflectionClass = new ReflectionClass($className);
    $modelInstance = $reflectionClass->newInstanceWithoutConstructor();

    // 1. Get columns from the $fillable property
    $fillableProperties = $reflectionClass->getDefaultProperties()['fillable'] ?? [];

    $columns = [];
    foreach ($fillableProperties as $prop) {
        $type = 'string'; // Default type
        if (str_ends_with($prop, '_at')) $type = 'timestamp';
        elseif (str_starts_with($prop, 'is_') || str_starts_with($prop, 'has_')) $type = 'boolean';
        elseif (str_ends_with($prop, '_id')) $type = 'foreignId';
        elseif (in_array($prop, ['description', 'content', 'body', 'details', 'notes'])) $type = 'text';

        $columns[] = ['name' => $prop, 'type' => $type, 'index' => ($type === 'foreignId')];
    }

    // 2. Get foreign keys from BelongsTo relationships
    $methods = $reflectionClass->getMethods(ReflectionMethod::IS_PUBLIC);
    foreach ($methods as $method) {
        if ($method->getNumberOfRequiredParameters() > 0) continue;

        $returnType = $method->getReturnType();
        if ($returnType && $returnType instanceof ReflectionNamedType) {
            if (str_ends_with($returnType->getName(), 'BelongsTo')) {
                // A BelongsTo relation implies a foreign key column on *this* model's table
                $relationName = $method->getName();
                $foreignKey = Illuminate\Support\Str::snake($relationName) . '_id';

                // Avoid adding duplicates if already found via $fillable
                $exists = false;
                foreach ($columns as $column) {
                    if ($column['name'] === $foreignKey) {
                        $exists = true;
                        break;
                    }
                }
                if (!$exists) {
                    $columns[] = ['name' => $foreignKey, 'type' => 'foreignId', 'index' => true];
                }
            }
        }
    }

    echo json_encode(['columns' => $columns], JSON_PRETTY_PRINT);

} catch (ReflectionException $e) {
    echo json_encode(['error' => "Reflection error: " . $e->getMessage()]);
    exit(1);
}
