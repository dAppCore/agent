---
name: /core:scaffold
description: Generate boilerplate code following Host UK patterns.
---

This command generates boilerplate code for models, actions, controllers, and modules.

## Subcommands

- `/core:scaffold model <name>` - Generate a Laravel model.
- `/core:scaffold action <name>` - Generate an Action class.
- `/core:scaffold controller <name>` - Generate an API controller.
- `/core:scaffold module <name>` - Generate a full module.

## `/core:scaffold model <name>`

Generates a new model file.

```php
<?php

declare(strict_types=1);

namespace Core\Models;

use Core\Tenant\Traits\BelongsToWorkspace;
use Illuminate\Database\Eloquent\Model;

class {{name}} extends Model
{
    use BelongsToWorkspace;

    protected $fillable = [
        'name',
        'email',
    ];
}
```


## `/core:scaffold action <name>`

Generates a new action file.

```php
<?php

declare(strict_types=1);

namespace Core\Actions;

use Core\Models\{{model}};
use Core\Support\Action;

class {{name}}
{
    use Action;

    public function handle(array $data): {{model}}
    {
        return {{model}}::create($data);
    }
}
```


## `/core:scaffold controller <name>`

Generates a new API controller file.

```php
<?php

declare(strict_types=1);

namespace Core\Http\Controllers\Api;

use Illuminate\Http\Request;
use Core\Http\Controllers\Controller;

class {{name}} extends Controller
{
    public function index()
    {
        //
    }

    public function store(Request $request)
    {
        //
    }

    public function show($id)
    {
        //
    }

    public function update(Request $request, $id)
    {
        //
    }

    public function destroy($id)
    {
        //
    }
}
```


## `/core:scaffold module <name>`

Generates a new module structure.

### `core-{{name}}/src/Core/Boot.php`
```php
<?php

declare(strict_types=1);

namespace Core\{{studly_name}}\Core;

class Boot
{
    // Boot the module
}
```

### `core-{{name}}/src/Core/ServiceProvider.php`
```php
<?php

declare(strict_types=1);

namespace Core\{{studly_name}}\Core;

use Illuminate\Support\ServiceProvider as BaseServiceProvider;

class ServiceProvider extends BaseServiceProvider
{
    public function register()
    {
        //
    }

    public function boot()
    {
        //
    }
}
```

### `core-{{name}}/composer.json`
```json
{
    "name": "host-uk/core-{{name}}",
    "description": "The Host UK {{name}} module.",
    "license": "EUPL-1.2",
    "authors": [
        {
            "name": "Claude",
            "email": "claude@host.uk.com"
        }
    ],
    "require": {
        "php": "^8.2"
    },
    "autoload": {
        "psr-4": {
            "Core\\{{studly_name}}\\": "src/"
        }
    },
    "config": {
        "sort-packages": true
    },
    "minimum-stability": "dev",
    "prefer-stable": true
}
```

### `core-{{name}}/CLAUDE.md`
```md
# Claude Instructions for `core-{{name}}`

This file provides instructions for the Claude AI agent on how to interact with the `core-{{name}}` module.
```

### `core-{{name}}/src/Mod/`

### `core-{{name}}/database/`

### `core-{{name}}/routes/`

### `core-{{name}}/tests/`
