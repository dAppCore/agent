<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Configs;

use Core\Config\Config;
use Core\Mod\Agentic\Services\AgenticManager;
use Illuminate\Validation\Rule;

/**
 * AI provider configuration.
 *
 * Manages AI service provider selection and custom instructions
 * for content generation in SocialHost.
 */
class AIConfig extends Config
{
    /**
     * Get the configuration group name.
     */
    public function group(): string
    {
        return 'ai';
    }

    /**
     * Get form field definitions with default values.
     *
     * @return array<string, mixed>
     */
    public function form(): array
    {
        return [
            'provider' => '',
            'instructions' => '',
        ];
    }

    /**
     * Get validation rules for form fields.
     *
     * @return array<string, mixed>
     */
    public function rules(): array
    {
        return [
            'provider' => [
                'sometimes',
                'nullable',
                Rule::in($this->getAvailableProviders()),
            ],
            'instructions' => ['sometimes', 'nullable', 'string', 'max:1000'],
        ];
    }

    /**
     * Get custom validation messages.
     *
     * @return array<string, string>
     */
    public function messages(): array
    {
        return [
            'provider.in' => 'The selected AI provider is not available.',
            'instructions.max' => 'Instructions may not be greater than 1000 characters.',
        ];
    }

    /**
     * Get list of available AI provider keys.
     *
     * @return array<int, string>
     */
    private function getAvailableProviders(): array
    {
        $agenticManager = app(AgenticManager::class);

        return array_keys($agenticManager->availableProviders());
    }
}
